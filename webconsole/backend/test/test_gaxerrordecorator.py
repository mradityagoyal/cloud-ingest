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
"""Unit tests for gaxerrordecorator.py."""

import unittest
from test.testutils import get_rpc_error_with_status_code
# Disable pylint relative-import since pylint bug makes pylint think google.gax
# is a relative import. Fix has been merged and will be included in
# next version of pylint (current version 1.7.2).
from google.gax import GaxError # pylint: disable=relative-import
from google.cloud.exceptions import Forbidden
from google.cloud.exceptions import NotFound
from google.cloud.exceptions import Conflict
from google.cloud.exceptions import Unauthorized
from grpc import StatusCode
from gaxerrordecorator import handle_common_gax_errors

class TestGaxErrorDecorator(unittest.TestCase):
    """Tests for gaxerrordecorator.py

    Since the gax error decorator is meant to be used only for SpannerWrapper
    methods, a FakeSelf class is defined to supply the self attributes read
    by the decorator. The tests place the decorator on methods that raise
    errors that are like those raised by the spanner python client library
    and ensure that the decorator handles them and raises the
    expected exception.
    """
    class FakeSelf(object):
        """Class used to mock the spannerwrapper class attributes."""
        # pylint: disable=too-few-public-methods
        project_id = "id"
        instance_id = "instance"
        database_id = "database"

    def setUp(self):
        """Set up for the tests"""
        self.fake_self = self.FakeSelf()

    def test_handles_permission_denied(self):
        """Test that the decorator correctly handles permission denied errors.
        """
        rpc_error = get_rpc_error_with_status_code(StatusCode.PERMISSION_DENIED)

        @handle_common_gax_errors
        def raise_permission_denied(fake_self):
            # pylint: disable=missing-docstring, unused-argument
            raise GaxError("msg", rpc_error)

        self.assertRaises(Forbidden, raise_permission_denied, self.fake_self)

    def test_handles_not_found(self):
        """Test that the decorator correctly handles not found errors."""
        rpc_error = get_rpc_error_with_status_code(StatusCode.NOT_FOUND)

        @handle_common_gax_errors
        def raise_not_found(fake_self):
            # pylint: disable=missing-docstring, unused-argument
            raise GaxError("msg", rpc_error)

        self.assertRaises(NotFound, raise_not_found, self.fake_self)

    def test_handles_unauthenticated(self):
        """Test that the decorator correctly handles unauthenticated errors."""
        rpc_error = get_rpc_error_with_status_code(StatusCode.UNAUTHENTICATED)

        @handle_common_gax_errors
        def raise_unauthenticated(fake_self):
            # pylint: disable=missing-docstring, unused-argument
            raise GaxError("msg", rpc_error)

        self.assertRaises(Unauthorized, raise_unauthenticated, self.fake_self)

    def test_handles_already_exists(self):
        """Test that the decorator correctly handles already exists errors."""
        rpc_error = get_rpc_error_with_status_code(StatusCode.ALREADY_EXISTS)

        @handle_common_gax_errors
        def raise_already_exists(fake_self):
            # pylint: disable=missing-docstring, unused-argument
            raise GaxError("msg", rpc_error)

        self.assertRaises(Conflict, raise_already_exists, self.fake_self)
