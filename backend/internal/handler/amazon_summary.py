import io
import json
import re
import sys
from datetime import datetime
from decimal import Decimal

import pdfplumber


def parse(text):
    display_names = [value.strip() for value in re.findall(r'^\s*Display\s+name\s*:?\s*(.*?)\s*$', text, re.IGNORECASE | re.MULTILINE) if value.strip()]
    if not display_names:
        lines = text.splitlines()
        for index, line in enumerate(lines):
            if re.fullmatch(r'\s*Display\s+name\s*', line, re.IGNORECASE):
                for candidate in lines[index + 1:]:
                    if candidate.strip():
                        display_names.append(candidate.strip())
                        break
                break
    if len(display_names) != 1 or not display_names[0].strip():
        raise ValueError('未识别到唯一的 Display name')
    display_name = display_names[0].strip()
    period = re.findall(r'Account activity from ([A-Za-z]+ \d{1,2}, \d{4}) 00:00 ([A-Z]+) through ([A-Za-z]+ \d{1,2}, \d{4}) 23:59 ([A-Z]+)', text)
    if len(period) != 1:
        raise ValueError('未识别到唯一且完整的报告日期范围')
    start, zone, end, end_zone = period[0]
    start = datetime.strptime(start, '%b %d, %Y').date()
    end = datetime.strptime(end, '%b %d, %Y').date()
    if start > end or zone not in ('PDT', 'PST') or end_zone not in ('PDT', 'PST'):
        raise ValueError('报告日期范围或时区无效')
    currencies = re.findall(r'All amounts in ([A-Z]{3}), unless specified', text, re.IGNORECASE)
    if len(currencies) != 1:
        raise ValueError('未识别到唯一的汇总报告币种')
    currency = currencies[0].upper()
    def amount(*labels, default=None):
        values = []
        for label in labels:
            values.extend(re.findall(re.escape(label) + r'\s+(-?(?:\d{1,3}(?:,\d{3})+|\d+)(?:\.\d{2})?)(?=\s|$)', text, re.IGNORECASE))
        if not values and default is not None:
            occurrences = sum(len(re.findall(re.escape(label), text, re.IGNORECASE)) for label in labels)
            if occurrences == 1:
                return default
        if len(values) != 1:
            raise ValueError('金额缺失或不唯一：' + ' / '.join(labels))
        return Decimal(values[0].replace(',', ''))
    # US summaries use the longer label; Canadian summaries use the shorter
    # wording. Both refer to the same withheld-tax total.
    tax_labels = ('Amazon Obligated Tax and Regulatory Fee Withheld', 'Amazon obligated tax withheld')
    return dict(display_name=display_name, start=str(start), end=str(end), currency=currency,
                tax=str(amount(*tax_labels)),
                storage=str(amount('FBA inventory and inbound services fees', default=Decimal('0'))),
                delivery=str(amount('FBA transaction fees') + amount('FBA transaction fee refunds')))


if __name__ == '__main__':
    try:
        with pdfplumber.open(io.BytesIO(sys.stdin.buffer.read())) as pdf:
            if not 1 <= len(pdf.pages) <= 10:
                raise ValueError('汇总 PDF 必须为 1–10 页')
            result = parse('\n'.join(page.extract_text() or '' for page in pdf.pages))
        print(json.dumps(result, ensure_ascii=True))
    except Exception as error:
        print(json.dumps({'error': str(error)}, ensure_ascii=True))
        sys.exit(1)
