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
"""Unit tests for main.py

Includes unit tests for the flask routes on main.py.
"""
import unittest
import main
from mock import patch
import json
from google.cloud.exceptions import Conflict
from google.cloud.exceptions import NotFound
from google.cloud.exceptions import Forbidden
from google.cloud.exceptions import PreconditionFailed
from google.cloud.exceptions import Unauthorized
from google.cloud.exceptions import BadRequest

class TestMain(unittest.TestCase):
    """Tests for main.py

    Includes tests for the routes in main.py.
    """
    # pylint: disable=too-many-public-methods

    def setUp(self):
        self.app = main.APP.test_client()
        self.app.testing = True

    @patch.object(main, '_get_credentials')
    def test_error_includes_traceback(self, _get_credentials_mock):
        """Tests that the common errors in main.py routes include a traceback"""
        def raise_exception_with_message(exception_class, message):
            """Raises the input exception class with input message."""
            raise exception_class(message)
        exception_list = [RuntimeError, BadRequest, Conflict, NotFound,
            Forbidden, PreconditionFailed, Unauthorized]
        expected_response_codes = [500, 400, 409, 404, 403, 412, 401]
        for i in range(0, len(exception_list)):
            def side_effect_function():
                """Side effect function for the mock used below"""
                raise_exception_with_message(exception_list[i], 'fake message')
            _get_credentials_mock.side_effect = side_effect_function
            response = self.app.get('/projects/fakeprojectid/jobconfigs')
            response_json = json.loads(response.data)
            self.assertEqual(response.status_code, expected_response_codes[i])
            self.assertTrue('fake message' in response_json['message'])
            self.assertTrue('Traceback' in response_json['traceback'])
            self.assertTrue('in raise_exception_with_message' in
              response_json['traceback'])
            self.assertTrue(exception_list[i].__name__ in
              response_json['traceback'])

if __name__ == '__main__':
    unittest.main()
