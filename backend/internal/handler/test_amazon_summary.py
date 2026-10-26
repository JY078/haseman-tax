import unittest
from amazon_summary import parse


class SummaryTests(unittest.TestCase):
    text = '''Display name: sunoutly
Account activity from Jul 1, 2026 00:00 PDT through Sep 30, 2026 23:59 PDT
All amounts in USD, unless specified
Amazon Obligated Tax and Regulatory Fee Withheld -11,887.64
FBA inventory and inbound services fees -24,785.46
FBA transaction fees -26,758.00
FBA transaction fee refunds 12.50
'''

    def test_values_and_refunds(self):
        result = parse(self.text)
        self.assertEqual(result['display_name'], 'sunoutly')
        self.assertEqual(result['start'], '2026-07-01')
        self.assertEqual(result['end'], '2026-09-30')
        self.assertEqual(result['tax'], '-11887.64')
        self.assertEqual(result['storage'], '-24785.46')
        self.assertEqual(result['delivery'], '-26745.50')

    def test_non_usd_currency_is_accepted(self):
        result = parse(self.text.replace('USD', 'CAD'))
        self.assertEqual(result['currency'], 'CAD')

    def test_missing_duplicate_currency_and_period(self):
        for text in [self.text.replace('Display name: sunoutly\n', ''),
                     self.text.replace('00:00', '12:00'),
                     self.text.replace('FBA transaction fee refunds 12.50', ''),
                     self.text + 'FBA transaction fees 10.00\n']:
            with self.subTest(text=text), self.assertRaises(ValueError):
                parse(text)

    def test_canadian_summary_labels_and_blank_zero_storage(self):
        text = '''Display name: QFN
Account activity from Jul 1, 2026 00:00 PDT through Jul 31, 2026 23:59 PDT
All amounts in CAD, unless specified
Amazon obligated tax withheld -1,011.55
FBA inventory and inbound services fees
FBA transaction fees -1,806.18
FBA transaction fee refunds 0
'''
        result = parse(text)
        self.assertEqual(result['currency'], 'CAD')
        self.assertEqual(result['tax'], '-1011.55')
        self.assertEqual(result['storage'], '0')
        self.assertEqual(result['delivery'], '-1806.18')


if __name__ == '__main__':
    unittest.main()
