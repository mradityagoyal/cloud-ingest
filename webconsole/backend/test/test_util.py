"""Contains the test functions for the utility module.
"""

import unittest
import json

from spannerwrapper import SpannerWrapper
from util import dict_has_values_recursively
from util import dict_values_are_recursively
from util import json_to_dictionary_in_field

# pylint: disable=too-many-public-methods,invalid-name

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

    def test_dict_values_are_recursively_true(self):
        """Tests dict_values_are_recursively method returns true."""

        statuses = {
            'cloudFunctionsStatus': 'NOT_FOUND',
            'dcpStatus': 'NOT_FOUND',
            'pubsubStatus': {
                'list': 'NOT_FOUND',
                'listProgress': 'NOT_FOUND',
                'loadBigQuery': 'NOT_FOUND',
                'loadBigQueryProgress': 'NOT_FOUND',
                'uploadGCS': 'NOT_FOUND',
                'uploadGCSProgress': 'NOT_FOUND'
            },
            'spannerStatus': 'NOT_FOUND'
        }
        self.assertTrue(dict_values_are_recursively(statuses, 'NOT_FOUND'))

        statuses['pubsubStatus']['loadBigQueryProgress'] = 'RUNNING'
        self.assertFalse(dict_values_are_recursively(statuses, 'NOT_FOUND'))

        statuses['pubsubStatus']['loadBigQueryProgress'] = 'NOT_FOUND'
        statuses['dcpStatus'] = 'RUNNING'
        self.assertFalse(dict_values_are_recursively(statuses, 'NOT_FOUND'))

    def test_dict_values_are_recursively_false(self):
        """Tests dict_values_are_recursively method returns false."""
        statuses = {
            "cloudFunctionsStatus": "RUNNING",
            "dcpStatus": "UNKNOWN",
            "pubsubStatus": {
                "list": "RUNNING",
                "listProgress": "RUNNING",
                "loadBigQuery": "RUNNING",
                "loadBigQueryProgress": "RUNNING",
                "uploadGCS": "RUNNING",
                "uploadGCSProgress": "RUNNING"
            },
            "spannerStatus": "RUNNING"
        }
        self.assertFalse(dict_values_are_recursively(statuses, 'NOT_FOUND'))

    def test_dict_values_are_recursively_one_false(self):
        """Tests dict_values_are_recursively method returns false when there
        is exactly one value that does not match.
        """

        statuses = {
            'cloudFunctionsStatus': 'RUNNING',
            'dcpStatus': 'NOT_FOUND',
            'pubsubStatus': {
                'list': 'NOT_FOUND',
                'listProgress': 'NOT_FOUND',
                'loadBigQuery': 'NOT_FOUND',
                'loadBigQueryProgress': 'NOT_FOUND',
                'uploadGCS': 'NOT_FOUND',
                'uploadGCSProgress': 'NOT_FOUND'
            },
            'spannerStatus': 'NOT_FOUND'
        }
        self.assertFalse(dict_values_are_recursively(statuses, 'NOT_FOUND'))

        statuses['cloudFunctionsStatus'] = 'NOT_FOUND'
        statuses['pubsubStatus']['loadBigQuery'] = 'RUNNING'
        self.assertFalse(dict_values_are_recursively(statuses, 'NOT_FOUND'))

    def test_dict_has_values_recursively_false(self):
        """Tests dict_has_values_recursively method returns false."""

        statuses = {
            'cloudFunctionsStatus': 'RUNNING',
            'dcpStatus': 'UNKNOWN',
            'pubsubStatus': {
                'list': 'NOT_FOUND',
                'listProgress': 'RUNNING',
                'loadBigQuery': 'UNKNOWN',
                'loadBigQueryProgress': 'NOT_FOUND',
                'uploadGCS': 'RUNNING',
                'uploadGCSProgress': 'UNKNOWN'
            },
            'spannerStatus': 'NOT_FOUND'
        }
        self.assertFalse(dict_has_values_recursively(
            statuses, set(['DEPLOYING', 'DELETING'])))

    def test_dict_has_values_recursively_true(self):
        """Tests dict_has_values_recursively method returns true."""

        statuses = {
            'cloudFunctionsStatus': 'RUNNING',
            'dcpStatus': 'DEPLOYING',
            'pubsubStatus': {
                'list': 'NOT_FOUND',
                'listProgress': 'RUNNING',
                'loadBigQuery': 'DELETING',
                'loadBigQueryProgress': 'NOT_FOUND',
                'uploadGCS': 'RUNNING',
                'uploadGCSProgress': 'UNKNOWN'
            },
            'spannerStatus': 'NOT_FOUND'
        }
        self.assertTrue(dict_has_values_recursively(
            statuses, set(['DEPLOYING', 'DELETING'])))

    def test_dict_has_values_recursively_one_true(self):
        """Tests dict_has_values_recursively method returns true when there is
        exactly one value that matches."""

        statuses = {
            'cloudFunctionsStatus': 'RUNNING',
            'dcpStatus': 'DEPLOYING',
            'pubsubStatus': {
                'list': 'NOT_FOUND',
                'listProgress': 'RUNNING',
                'loadBigQuery': 'NOT_FOUND',
                'loadBigQueryProgress': 'NOT_FOUND',
                'uploadGCS': 'RUNNING',
                'uploadGCSProgress': 'UNKNOWN'
            },
            'spannerStatus': 'NOT_FOUND'
        }
        self.assertTrue(dict_has_values_recursively(
            statuses, set(['DEPLOYING', 'DELETING'])))

        statuses['dcpStatus'] = 'RUNNING'
        statuses['pubsubStatus']['listProgress'] = 'DELETING'
        self.assertTrue(dict_has_values_recursively(
            statuses, set(['DEPLOYING', 'DELETING'])))


if __name__ == '__main__':
    unittest.main()
