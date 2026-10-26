import os
import unittest
from types import SimpleNamespace
from unittest.mock import patch

from parse import field_key, parse_pdf, read_fields, read_items, trade_mode_value


def word(text, center, top, width=12):
    return {'text': text, 'x0': center - width / 2, 'x1': center + width / 2, 'top': top, 'bottom': top + 8}


class CustomsParserTests(unittest.TestCase):
    def test_trade_mode_retains_code_from_label(self):
        page = SimpleNamespace(width=595, extract_tables=lambda: [[['贸易方式（9810）\n跨境电商出口海外仓']]])
        self.assertEqual(read_fields(page, [], False)['trade_mode'], '跨境电商出口海外仓(9810)')
        self.assertEqual(trade_mode_value('监管方式 ( 0110 )', '一般贸易'), '一般贸易(0110)')
        self.assertEqual(trade_mode_value('贸易方式(9810)', '(9810) 跨境电商出口海外仓'), '跨境电商出口海外仓(9810)')
        self.assertEqual(trade_mode_value('贸易方式(9810)', '跨境电商出口海外仓（9810）'), '跨境电商出口海外仓(9810)')
        self.assertEqual(trade_mode_value('', '(0110) 一般贸易'), '一般贸易(0110)')
        self.assertEqual(trade_mode_value('贸易方式', '一般贸易'), '一般贸易')

    def test_trade_mode_code_from_ocr_words(self):
        page = SimpleNamespace(width=595)
        words = [word('贸易方式', 70, 100), word('(9810)', 120, 100), word('跨境电商出口海外仓', 110, 120)]
        self.assertEqual(read_fields(page, words, True)['trade_mode'], '跨境电商出口海外仓(9810)')

    def test_header_labels_and_blank_cells(self):
        page = SimpleNamespace(width=595, extract_tables=lambda: [[['备案号', '境内收发货人(123)\n测试公司', '运费\nUSD/12.30/总价']]])
        self.assertEqual(read_fields(page, [], False), {'record_number': '', 'consignor': '测试公司', 'freight': 'USD/12.30/总价'})
        self.assertIsNone(field_key('商品序号 商品编号'))

    def test_multiple_items_and_quantity_continuations(self):
        names = ['商品序号', '商品编号', '商品名称、规格型号', '数量/单位', '数量及单位', '目的国(地区)', '单价', '总价', '币制', '征免']
        words = [word(name, 30 + i * 50, 100) for i, name in enumerate(names)]
        for top, values in [(115, ['1', '8481809000', '水龙头', '法定数量/法定单位', '120套', '加拿大', '15.1597', '1819.16', 'USD', '']),
                            (130, ['', '', '无品牌|无型号', '第二数量/第二单位', '216千克', '', '', '', '', '']),
                            (145, ['', '', '', '申报数量/申报单位', '120套', '', '', '', '', '']),
                            (165, ['2', '8481809001', '配件', '法定数量/法定单位', '2套', '加拿大', '1.25', '2.50', 'USD', ''])]:
            words.extend(word(value, 30 + i * 50, top) for i, value in enumerate(values) if value)
        words.append(word('税费征收情况', 30, 200))
        items = read_items(words)
        self.assertEqual(len(items), 2)
        self.assertEqual(items[0]['declared_quantity'], '120套')
        self.assertEqual(items[0]['secondary_quantity'], '216千克')
        self.assertEqual(items[0]['specification'], '无品牌|无型号')
        self.assertEqual(items[1]['total_price'], '2.50')

    def document(self, texts):
        pages = [SimpleNamespace(extract_text=lambda text=text: text, extract_words=lambda: [], extract_tables=lambda: [], width=595) for text in texts]
        context = unittest.mock.MagicMock()
        context.__enter__.return_value.pages = pages
        return context

    def test_reject_different_declarations_in_one_file(self):
        with patch('parse.pdfplumber.open', return_value=self.document(['中华人民共和国海关出口货物报关单 海关编号：310120260000665636 ' * 2, '报关单 海关编号：310120260000665637 ' * 3])):
            with self.assertRaisesRegex(ValueError, '唯一'):
                parse_pdf('unused.pdf')

    def test_missing_items_report_warning(self):
        with patch('parse.pdfplumber.open', return_value=self.document(['中华人民共和国海关出口货物报关单 海关编号：310120260000665636 ' * 2])):
            result = parse_pdf('unused.pdf')
            self.assertEqual(result['declaration_number'], '310120260000665636')
            self.assertTrue(any('商品明细' in warning for warning in result['warnings']))

    @unittest.skipUnless(os.getenv('CUSTOMS_SAMPLE_PDF'), 'set CUSTOMS_SAMPLE_PDF to check supplied PDF')
    def test_actual_sample(self):
        result = parse_pdf(os.environ['CUSTOMS_SAMPLE_PDF'])
        self.assertEqual(result['declaration_number'], '310120260000665636')
        self.assertEqual(result['declaration_date'], '2026-09-18')
        self.assertEqual(result['fields']['contract_number'], 'FUS-26696')
        self.assertEqual(result['fields']['gross_weight'], '228')
        self.assertEqual(len(result['items']), 1)
        self.assertEqual(result['items'][0]['declared_quantity'], '120套')
        self.assertEqual(result['items'][0]['total_price'], '1819.16')
        self.assertEqual(result['items'][0]['currency'], 'USD')
        self.assertEqual(result['warnings'], [])


if __name__ == '__main__':
    unittest.main()
