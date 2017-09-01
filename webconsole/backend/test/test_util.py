"""Contains the test functions for the utility module.
"""

import unittest
import json
from util import json_to_dictionary_in_field
from spannerwrapper import SpannerWrapper

JOB_SPEC_1 = {
    u'gcsDirectory': u'/fake/gcs/directory',
    u'onPremSrcDirectory': u'/fake/on/prem/source',
    u'gcsBucket': u'fakeGCSBucket',
    u'bigqueryDataset': u'fakeBigqueryDataset',
    u'bigqueryTable': u'fakeBigqueryTable'
}

JOB_SPEC_2 = {
    u'gcsDirectory': u'/fake/gcs/directory2',
    u'onPremSrcDirectory': u'/fake/on/prem/source2',
    u'gcsBucket': u'fakeGCSBucket2',
    u'bigqueryDataset': u'fakeBigqueryDataset2',
    u'bigqueryTable': u'fakeBigqueryTable2'
}

JOB_SPEC_3 = {
    u'gcsDirectory': u'/fake/gcs/directory3',
    u'onPremSrcDirectory': u'/fake/on/prem/source3',
    u'gcsBucket': u'fakeGCSBucket3',
    u'bigqueryDataset': u'fakeBigqueryDataset3',
    u'bigqueryTable': u'fakeBigqueryTable3'
}

JOB_SPEC_1_STR = json.dumps(JOB_SPEC_1)

JOB_SPEC_2_STR = json.dumps(JOB_SPEC_2)

JOB_SPEC_3_STR = json.dumps(JOB_SPEC_3)


FAKE_JOB_CONFIG_LIST = [
    {
        u'JobSpec': JOB_SPEC_1_STR,
        u'JobConfigId': u'fakeConfig1'
    },
    {
        u'JobSpec': JOB_SPEC_2_STR,
        u'JobConfigId': u'fakeConfig2'
    },
        {
        u'JobSpec': JOB_SPEC_3_STR,
        u'JobConfigId': u'fakeConfig3'
    },
]

FAKE_JOB_CONFIG_OBJ_LIST = [
    {
        u'JobSpec': JOB_SPEC_1,
        u'JobConfigId': u'fakeConfig1'
    },
    {
        u'JobSpec': JOB_SPEC_2,
        u'JobConfigId': u'fakeConfig2'
    },
        {
        u'JobSpec': JOB_SPEC_3,
        u'JobConfigId': u'fakeConfig3'
    },
]

class TestUtil(unittest.TestCase):
    """
    Contains unit tests for the utility module.
    """
    def test_json_to_dictionary_in_field(self):
        """
        Tests the json_to_dictionary_in_field function.
        """
        actual_list = json_to_dictionary_in_field(FAKE_JOB_CONFIG_LIST,
            SpannerWrapper.JOB_SPEC)
        self.assertEqual(actual_list, FAKE_JOB_CONFIG_OBJ_LIST)


if __name__ == '__main__':
    unittest.main()
