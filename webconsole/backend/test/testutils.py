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
from spannerwrapper import SpannerWrapper

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

# pylint: disable=too-many-arguments
# Many arguments are needed to create a task.
def get_task(config_id, run_id, task_id, task_creation_time, last_mod_time,
             status, task_spec, task_type):
    """Returns a task in dictionary format containing the given values.

    Args:
      config_id: The job config id of the task
      run_id: The job run id of the task
      task_id: The id of the task
      task_creation_time: The creation time
      last_mod_time: An int representing the last modification time
      status: An int representing the status of the task
      task_spec: The task spec, a JSON string

    Returns:
      A task in dictionary format with the given values.
    """
    return {
        SpannerWrapper.JOB_CONFIG_ID: config_id,
        SpannerWrapper.JOB_RUN_ID: run_id,
        SpannerWrapper.TASK_ID: task_id,
        SpannerWrapper.TASK_CREATION_TIME: task_creation_time,
        SpannerWrapper.LAST_MODIFICATION_TIME: last_mod_time,
        SpannerWrapper.STATUS: status,
        SpannerWrapper.TASK_SPEC: task_spec,
        SpannerWrapper.TASK_TYPE: task_type,
    }
