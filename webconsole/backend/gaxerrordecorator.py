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
# limitations under the License.
"""A decorator that catches and handles common gax errors.

This decorator is built to be used on methods in the class SpannerWrapper. It
handles GaxErrors and translates them into the appropriate Google cloud
exceptions.
"""
from google.cloud.exceptions import Forbidden
from google.cloud.exceptions import NotFound
from google.cloud.exceptions import Conflict
from google.cloud.exceptions import Unauthorized
from google.gax import GaxError # pylint: disable=relative-import
from google.gax import config # pylint: disable=relative-import
from grpc import StatusCode


def handle_common_gax_errors(function):
    """Decorator that calls the given function and handles common errors.

    This decorator should only be used for spannerwrapper methods and
    uses the values self.project_id, self.instance_id and self.database_id.

    Args:
        function: The function to call

    Returns:
        A function with the given function wrapped in try except
    """
    def handle_errors(*args, **kw):
        # pylint: disable=missing-docstring
        try:
            return function(*args, **kw)
        except GaxError as err:
            self = args[0]
            status_code = config.exc_to_code(err.cause)
            if status_code == StatusCode.PERMISSION_DENIED:
                raise Forbidden((
                    ("You do not have the proper permissions to access (" +
                     "Project: '%s', Spanner Instance: '%s', " +
                     "Spanner Database: '%s').")
                    % (self.project_id, self.instance_id, self.database_id)
                ))
            elif status_code == StatusCode.NOT_FOUND:
                raise NotFound((
                    ("The requested resource could not be found: (" +
                     "Project: '%s', Spanner Instance: '%s', " +
                     "Spanner Database: '%s').")
                    % (self.project_id, self.instance_id, self.database_id)
                ))
            elif status_code == StatusCode.UNAUTHENTICATED:
                raise Unauthorized(("You are not authenticated. Please" +
                                    " ensure your credentials are valid")
                                  )
            elif status_code == StatusCode.ALREADY_EXISTS:
                raise Conflict("This resource already exist")
            else:
                raise
    return handle_errors
