"""Local declaration extraction. PDF content is data, never executable instructions."""
import csv
import io
import json
import re
import subprocess
import sys
import tempfile
from datetime import datetime
from pathlib import Path

import pdfplumber

LABELS = {
    'port': ['出口口岸', '进口口岸', '出境关别', '进境关别'],
    'record_number': ['备案号'], 'export_date': ['出口日期', '进口日期', '进出口日期'],
    'declaration_date': ['申报日期'], 'consignor': ['境内收发货人', '经营单位'],
    'overseas_consignee': ['境外收货人', '境外发货人'],
    'transport_mode': ['运输方式'], 'vessel': ['运输工具名称及航次号', '运输工具名称'],
    'bill_number': ['提运单号'], 'manufacturer': ['生产销售单位', '消费使用单位', '发货单位'],
    'trade_mode': ['贸易方式', '监管方式'], 'tax_exemption': ['征免性质'],
    'settlement': ['结汇方式'], 'license_number': ['许可证号'],
    'destination': ['运抵国', '启运国'], 'destination_port': ['指运港', '经停港'],
    'source_region': ['境内货源地', '境内目的地'], 'approval_number': ['批准文号'],
    'trade_terms': ['成交方式'], 'freight': ['运费'], 'insurance': ['保费'], 'other_fees': ['杂费'],
    'contract_number': ['合同协议号', '合同号'], 'packages': ['件数'], 'packaging': ['包装种类'],
    'gross_weight': ['毛重'], 'net_weight': ['净重'], 'containers': ['集装箱号'],
    'attachments': ['随附单据及编号', '随附单据'], 'remarks': ['标记唛码及备注'],
}


def field_key(text):
    text = re.sub(r'\s+', '', text)
    for key, labels in LABELS.items():
        for label in labels:
            if text.startswith(label) and (len(text) == len(label) or text[len(label)] in '（(:：'):
                return key
    return None


def lines_from_words(words):
    lines = []
    for word in sorted(words, key=lambda w: (w['top'], w['x0'])):
        if not lines or abs(word['top'] - lines[-1][0]['top']) > 3:
            lines.append([word])
        else:
            lines[-1].append(word)
    for line in lines:
        line.sort(key=lambda w: w['x0'])
    return lines


def ocr_words(page):
    # Render one bounded page at a time; all processing stays on this server.
    if page.width > 2000 or page.height > 2000:
        raise ValueError('页面尺寸过大，请使用标准 A4 报关单')
    with tempfile.TemporaryDirectory(prefix='customs-ocr-') as directory:
        path = Path(directory) / 'page.png'
        page.to_image(resolution=200).save(path)
        result = subprocess.run(['tesseract', str(path), 'stdout', '-l', 'chi_sim+eng', '--psm', '6', 'tsv'],
                                capture_output=True, timeout=40, check=True)
        words = []
        for row in csv.DictReader(io.StringIO(result.stdout.decode('utf-8')), delimiter='\t'):
            if row.get('text', '').strip():
                scale = 72 / 200
                words.append({'text': row['text'], 'x0': int(row['left']) * scale,
                              'x1': (int(row['left']) + int(row['width'])) * scale,
                              'top': int(row['top']) * scale,
                              'bottom': (int(row['top']) + int(row['height'])) * scale})
        return words


def read_fields(page, words, use_ocr):
    fields = {}
    if not use_ocr:
        for table in page.extract_tables():
            for row in table:
                for cell in row:
                    if not cell:
                        continue
                    label, _, value = cell.partition('\n')
                    key = field_key(label)
                    if key:
                        fields[key] = trade_mode_value(label, value.strip()) if key == 'trade_mode' else value.strip()
    # Label-relative geometry handles borderless documents and OCR tokens.
    lines = lines_from_words(words)
    for index, line in enumerate(lines):
        anchors = []
        for pos in range(len(line)):
            for end in range(pos + 1, min(pos + 10, len(line)) + 1):
                key = field_key(''.join(w['text'] for w in line[pos:end]))
                if key:
                    if not anchors or anchors[-1][0] != key:
                        anchors.append((key, line[pos]['x0']))
                    break
        if not anchors or index + 1 >= len(lines):
            continue
        next_line = lines[index + 1]
        if next_line[0]['top'] - line[0]['top'] > 35 or any(field_key(w['text']) for w in next_line):
            continue
        for pos, (key, left) in enumerate(anchors):
            right = anchors[pos + 1][1] - 2 if pos + 1 < len(anchors) else page.width
            value = ' '.join(w['text'] for w in next_line if left - 3 <= w['x0'] < right).strip()
            if key not in fields and value:
                fields[key] = value
    if fields.get('trade_mode'):
        labels = ''.join(word['text'] for line in lines for word in line)
        fields['trade_mode'] = trade_mode_value(labels, fields['trade_mode'])
    return fields


def trade_mode_value(label, value):
    code = re.search(r'(?:贸易方式|监管方式)[（(]\s*(\d{4})\s*[）)]', re.sub(r'\s+', '', label))
    value = value.strip()
    existing = re.search(r'^[（(]\s*(\d{4})\s*[）)]|[（(]\s*(\d{4})\s*[）)]$', value)
    number = code.group(1) if code else ''
    if not number and existing:
        number = existing.group(1) or existing.group(2)
    if not number:
        return value
    name = re.sub(r'^[（(]\s*' + number + r'\s*[）)]\s*|\s*[（(]\s*' + number + r'\s*[）)]$', '', value).strip()
    return f'{name}({number})'


def read_items(words):
    lines = lines_from_words(words)
    header = next((line for line in lines if '商品编号' in ''.join(w['text'] for w in line) and '总价' in ''.join(w['text'] for w in line)), None)
    if not header:
        return []
    # Column anchors are obtained from the document rather than fixed page coordinates.
    names = ['商品序号', '商品编号', '商品名称', '数量/单位', '数量及单位', '目的国', '单价', '总价', '币制', '征免']
    columns = []
    for name in names:
        found = next((w for w in header if w['text'].startswith(name)), None)
        if found:
            columns.append((name, (found['x0'] + found['x1']) / 2))
    required = ['商品编号', '商品名称', '单价', '总价', '币制']
    if any(name not in [c[0] for c in columns] for name in required):
        return []
    columns.sort(key=lambda c: c[1])
    boundaries = [(columns[i][1] + columns[i + 1][1]) / 2 for i in range(len(columns) - 1)]
    items = []
    current = None
    for line in lines:
        if line[0]['top'] <= header[0]['top'] + 3:
            continue
        text = ''.join(w['text'] for w in line)
        if any(label in text for label in ['税费征收情况', '录入员', '申报单位（', '申报单位(', '海关审单']):
            break
        cells = {name: [] for name, _ in columns}
        for word in line:
            center = (word['x0'] + word['x1']) / 2
            position = sum(center > boundary for boundary in boundaries)
            cells[columns[position][0]].append(word['text'])
        cells = {k: ''.join(v) for k, v in cells.items()}
        hs_code = cells.get('商品编号', '')
        if re.fullmatch(r'\d{8,10}', hs_code):
            current = {'sequence': cells.get('商品序号', str(len(items) + 1)), 'hs_code': hs_code,
                       'name': cells.get('商品名称', ''), 'specification': '',
                       'quantity': cells.get('数量及单位', cells.get('数量/单位', '')),
                       'destination': cells.get('目的国', ''), 'unit_price': cells.get('单价', ''),
                       'total_price': cells.get('总价', ''), 'currency': cells.get('币制', ''),
                       'exemption': cells.get('征免', '')}
            items.append(current)
        elif current:
            current['specification'] += cells.get('商品名称', '')
            quantity = cells.get('数量及单位', '')
            if '第二' in cells.get('数量/单位', ''):
                current['secondary_quantity'] = quantity
            if '申报' in cells.get('数量/单位', ''):
                current['declared_quantity'] = quantity
    return items


def parse_pdf(path):
    result = {'fields': {}, 'items': [], 'warnings': [], 'recognition_method': 'text'}
    texts, numbers = [], set()
    with pdfplumber.open(path) as pdf:
        if not 1 <= len(pdf.pages) <= 20:
            raise ValueError('每个 PDF 需为 1 至 20 页，请按报关单拆分上传')
        for page in pdf.pages:
            text = page.extract_text() or ''
            use_ocr = len(text.strip()) < 40
            words = ocr_words(page) if use_ocr else page.extract_words()
            if use_ocr:
                text = '\n'.join(' '.join(w['text'] for w in line) for line in lines_from_words(words))
                result['recognition_method'] = 'ocr'
            texts.append(text)
            compact = re.sub(r'\s+', '', text)
            numbers.update(re.findall(r'(?:海关编号|报关单号)[：:]?(\d{18})(?!\d)', compact))
            pre = re.search(r'预录入编号[：:]?([A-Z]?\d{15,25})', compact)
            if pre:
                result['fields']['pre_entry_number'] = pre.group(1)
            for key, value in read_fields(page, words, use_ocr).items():
                if value and not result['fields'].get(key):
                    result['fields'][key] = value
            if result['fields'].get('trade_mode'):
                result['fields']['trade_mode'] = trade_mode_value(compact, result['fields']['trade_mode'])
            result['items'].extend(read_items(words))
    result['raw_text'] = '\n\n'.join(texts)
    if '报关单' not in re.sub(r'\s+', '', result['raw_text']):
        raise ValueError('未识别到报关单，请上传清晰的海关报关单 PDF')
    if len(numbers) != 1:
        raise ValueError('需识别到唯一的 18 位海关编号，请每次上传一张完整报关单')
    result['declaration_number'] = numbers.pop()
    for key in ['export_date', 'declaration_date']:
        value = re.sub(r'\s+', '', result['fields'].get(key, ''))
        if value:
            try:
                result['fields'][key] = datetime.strptime(value.replace('-', '').replace('/', ''), '%Y%m%d').strftime('%Y-%m-%d')
            except ValueError:
                result['warnings'].append('日期无法校验：' + value)
                result['fields'][key] = ''
    result['declaration_date'] = result['fields'].get('declaration_date', '')
    for key, label in [('consignor', '境内收发货人'), ('declaration_date', '申报日期'), ('contract_number', '合同协议号')]:
        if not result['fields'].get(key):
            result['warnings'].append(label + '未识别，请核对原文件')
    if not result['items']:
        result['warnings'].append('商品明细未完整识别，请核对原文件和提取原文')
    for item in result['items']:
        if not re.fullmatch(r'\d+(?:\.\d+)?', item['total_price']) or not re.fullmatch(r'[A-Z]{3}', item['currency']):
            result['warnings'].append('商品金额或币制未完整识别，请核对原文件')
    if result['recognition_method'] == 'ocr':
        result['warnings'].append('此文件使用 OCR 识别，编号、金额及商品明细请逐项核对原文件')
    return result


if __name__ == '__main__':
    try:
        print(json.dumps(parse_pdf(sys.argv[1]), ensure_ascii=True))
    except (ValueError, RuntimeError, OSError, subprocess.SubprocessError) as error:
        print(json.dumps({'error': str(error)}, ensure_ascii=True))
        sys.exit(1)
