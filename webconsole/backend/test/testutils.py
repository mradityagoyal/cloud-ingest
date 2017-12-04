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
"""Util functions for the web console back-end python module tests."""

from grpc import RpcError

def get_rpc_error_with_status_code(status_code):
    """Returns an RpcError with the given status_code.

    Returns an RpcError with the given status_code stored in such a way
    that google.gax.config.exc_to_code can extract the status code.

    Returns:
        RpcError containing the status code
    """
    def return_code():
        """Returns the status code."""
        return status_code

    err = RpcError()
    # config.exc_to_code calls .code() on the exception to get the status code
    err.code = return_code
    return err
