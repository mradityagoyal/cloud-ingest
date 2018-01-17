# -*- coding: utf-8 -*-
# Copyright 2017 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License
"""Unit tests for spannerwrapper.py.

Tests that data is processed and returned in the proper format. Also
tests that creation methods return the appropriate value according to
the presence or lack of exceptions. The Cloud Spanner client library is
mocked, so these tests do not cover connecting to Cloud Spanner.
"""
import logging
import unittest

from mock import MagicMock
from mock import patch

from spannerwrapper import SpannerWrapper

# Disable pylint since pylint bug makes pylint think google.gax
# is a relative import. Fix has been merged and will be included in
# next version of pylint (current version 1.7.2).

PROJECT_ID = u'test-project'
JOB_CONFIG_ID_1 = u'test-config1'
JOB_CONFIG_ID_2 = u'test-config2'
JOB_SPEC_1 = {u'srcDir': u'usr/home/'}
JOB_SPEC_2 = {u'srcDir': u'usr/home2/'}
JOB_SPEC_STR_1 = '{"srcDir": "usr/home/"}'
JOB_SPEC_STR_2 = '{"srcDir": "usr/home2/"}'

class TestSpannerWrapper(unittest.TestCase):
    """Unit tests for spannerwrapper.py with the Cloud Spanner client mocked."""
    # pylint: disable=too-many-public-methods

    time_mock = MagicMock()
    time_mock.return_value = 12345

    @patch('spannerwrapper.spanner')
    # pylint: disable=arguments-differ
    def setUp(self, spanner_mock):
    # pylint: enable=arguments-differ
        logging.disable(logging.CRITICAL) # So tests don't print to console
        database = MagicMock()
        snapshot_obj = MagicMock()
        self.snapshot = MagicMock()

        self.spanner_instance = MagicMock()
        self.spanner_instance.database.return_value = database
        database.snapshot.return_value = snapshot_obj
        snapshot_obj.__enter__.return_value = self.snapshot

        self.spanner_client = MagicMock()
        self.spanner_client.instance.return_value = self.spanner_instance

        spanner_mock.Client.return_value = self.spanner_client

        self.pool = MagicMock()
        spanner_mock.BurstyPool.return_value = self.pool

        self.spanner_wrapper = SpannerWrapper('', '', '', '')

    def test_get_job_configs(self):
        """Asserts that two job configs are successfully returned."""
        result = MagicMock()
        result.__iter__.return_value = [
            [PROJECT_ID, JOB_CONFIG_ID_1, JOB_SPEC_STR_1],
            [PROJECT_ID, JOB_CONFIG_ID_2, JOB_SPEC_STR_2]]
        result.fields = self.get_fields_list(
            SpannerWrapper.JOB_CONFIGS_COLUMNS)
        self.snapshot.execute_sql.return_value = result

        actual = self.spanner_wrapper.get_job_configs(PROJECT_ID)
        expected = [{u'ProjectId': PROJECT_ID,
                     u'JobConfigId': JOB_CONFIG_ID_1,
                     u'JobSpec': JOB_SPEC_1},
                    {u'ProjectId': PROJECT_ID,
                     u'JobConfigId': JOB_CONFIG_ID_2,
                     u'JobSpec': JOB_SPEC_2}]
        self.assertEqual(actual, expected)

    def test_get_configs_nonexistent(self):
        """Asserts that an empty list is returned when there are no configs."""
        result = MagicMock()
        result.__iter__.return_value = []
        self.snapshot.execute_sql.return_value = result

        actual = self.spanner_wrapper.get_job_configs(PROJECT_ID)
        self.assertEqual(actual, [])

    def test_get_job_configs_table(self):
        """Asserts that the get_job_configs query uses the JobConfigs table."""
        self.spanner_wrapper.get_job_configs(PROJECT_ID)
        self.snapshot.execute_sql.assert_called()
        query = self.snapshot.execute_sql.call_args[0][0]
        self.assertIn(SpannerWrapper.JOB_CONFIGS_TABLE, query)

    @staticmethod
    def get_fields_list(fields):
        """Returns fields in the format returned by the Cloud Spanner client.

        Returns a list of objects with the name property populated with the
        given fields.

        Args:
          fields: A list of strings representing the names of the fields.

        Returns:
          A list of fields like that returned by the Cloud Spanner client.
        """
        mocks = []
        for field in fields:
            field_mock = MagicMock()
            field_mock.name = field
            mocks.append(field_mock)
        return mocks


if __name__ == '__main__':
    unittest.main()
