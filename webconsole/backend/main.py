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
"""Runs the back-end of the Ingest Web Console, an application written in Flask.

The application is set-up to be a RESTful API. Any uncaught exceptions raised
during the processing of a request will be handled by the error handler
for Internal Server Errors and a response of 500 Internal Server Error will
be sent.
"""
import argparse
import httplib
import json
import logging
import re
import traceback

from flask import Flask
from flask import jsonify
from flask import request
from google.cloud.exceptions import BadRequest
from google.cloud.exceptions import Conflict
from google.cloud.exceptions import Forbidden
from google.cloud.exceptions import NotFound
from google.cloud.exceptions import PreconditionFailed
from google.cloud.exceptions import ServerError
from google.cloud.exceptions import Unauthorized
from google.oauth2.credentials import Credentials
from proto.tasks_pb2 import TaskFailureType
from proto.tasks_pb2 import TaskStatus


import infra_util
import util
from corsdecorator import crossdomain  # To allow requests from the front-end
from spannerwrapper import SpannerWrapper

APP = Flask(__name__)
APP.config.from_pyfile('ingestwebconsole.default_settings')
APP.config.from_envvar('INGEST_CONFIG_PATH')

# Allowed headers in cross-site http requests.
_ALLOWED_HEADERS = ['Content-Type', 'Authorization']

_AUTH_HEADER_REGEX = re.compile(r'^\s*Bearer\s+(?P<access_token>[^\s]*)$')

# A regex pattern that matches a valid job config id (description).
_CONFIG_ID_PATTERN = re.compile('^[0-9A-Za-z _-]+$')

# The error returned to the user indicating that the config id is invalid.
_CONFIG_FORMAT_ERROR = {
    'error' : 'The job config id is not in the expected format',
    'message' : ('A job config id can only contain alphanumeric  characters,'
                 ' underscore, hyphen and space.')
    }

# A regex pattern that matches a valid project id. Matches a string that has
# lowercase letters, digits or hyphens and must start with a lowercase letter.
# It must be between 6 and 30 characters
_PROJECT_ID_PATTERN = re.compile(r'^[a-z]([0-9a-z-]){5,29}$')

# The error returned to the user indicating that the project id is in an
# invalid format.
_PROJECT_ID_FORMAT_ERROR = {
    'error' : 'The project id is not in the expected format',
    'message' : ('Project ID must be between 6 and 30 characters. A project ID '
                 ' can have lowercase letters, digits or hyphens and must start'
                 ' with a lowercase letter.')
    }

# A regex pattern that matches an absolute unix path.
_UNIX_SOURCE_PATH_PATTERN = re.compile(r'^(/[^/ ]*)+/?$')

# A regex pattern that matches an absolute windows path.
# Taken from:
# https://www.safaribooksonline.com/library/view/regular-expressions-cookbook/9781449327453/ch08s18.html
_WINDOWS_SOURCE_PATH_PATTERN = re.compile(
r"""\A
[a-z]:\\                    # Drive
(?:[^\\/:*?"<>|\r\n]+\\)*   # Folder
[^\\/:*?"<>|\r\n]*          # File
\Z""", re.VERBOSE | re.IGNORECASE)

# An error indicating that the path provided was not a valid unix or windows
# path.
_PATH_PATTERN_ERROR = {
    'error' : 'Invalid pattern',
    'message' : 'The source path is not a valid windows or unix pattern.'
}

# A regex pattern that matches a valid bucket id.
_BUCKET_PATTERN = re.compile(r'^[a-z0-9](\.|[0-9a-z_-])+[a-z0-9]$')

# An error determining that the bucket pattern
_BUCKET_PATTERN_ERRROR = {
    'error' : 'Invalid bucket name',
    'message' : 'The input bucket name is not a valid bucket.'
}


# An error returned to the user indicating that the job config id was not
# found.
_CONFIG_NOT_FOUND_ERROR = {
    'error' : 'Job Config ID',
    'message' : ('The input job config id was not found on the'
                    'database')
}

# An error returned to the user indicating that no configurations were found.
_NO_CONFIGS_ERROR = {
    'error' : 'No configurations found',
    'message': 'No configurations were found for the supplied project id.'
}

# An input string indicating the file system directory argument for jobs.
FILE_SYSTEM_DIR = 'fileSystemDirectory'

# An input string indicating the gcs bucket argument for jobs.
GCS_BUCKET = 'gcsBucket'

# An input string indicating the job config id argument for jobs.
CONFIG_ID = 'jobConfigId'

# A string indicating the input gcs directory argument for jobs.
ON_PREM_SRC_DIRECTORY = 'onPremSrcDirectory'

# The required fields for a post job config request.
POST_CONFIG_FIELDS = [FILE_SYSTEM_DIR, GCS_BUCKET, CONFIG_ID]

# An error returned to the user indicating that a field was missing on posting a
# job configuration.
_MISSING_FIELD_ERROR = {
    'error' : 'Missing required field',
    'message' : ('Missing at least one of the required fields: '
                    ','.join(POST_CONFIG_FIELDS))
}

# An error returned to the user indicating that the task status was not in the
# list of accepted task status.
_INVALID_TASK_STATUS_ERROR = {
    'error': 'Invalid task status',
    'message': 'The provided task status is not recognized.'
}

# An error returned to the user indicating that the failure type was not in the
# list of accepted task failure types.
_INVALID_TASK_FAILURE_TYPE_ERROR = {
    'error': 'Invalid task failure type',
    'message': 'The provided task failure type is not recognized.'
}

# An error returned to the user indicating that the last modified timestamp is
# invalid.
_INVALID_TIMESTAMP_ERROR = {
    'error' : 'Invalid last modified timestamp',
    'message' : 'The timestamp must be between 0 and now.'
}

def _get_credentials():
    """Gets OAuth2 credentials from request Authorization headers."""
    auth_header = request.headers.get('Authorization')
    if not auth_header:
        raise Unauthorized(('Missing Authorization header. Please add ' +
                            'Authorization header in the format "Bearer ' +
                            '<access_token>".'))
    match = _AUTH_HEADER_REGEX.match(auth_header)
    if not match:
        raise BadRequest(
            'Invalid Authorization header format, the Authorization header '
            'should be in the format "Bearer <access_token>".')
    access_token = match.group('access_token')

    credentials = Credentials(access_token)
    return credentials

def _get_post_job_configs_job_spec(content):
    """Makes a jobspec dictionary out of the input request content."""
    return {
        ON_PREM_SRC_DIRECTORY : content[FILE_SYSTEM_DIR].strip(),
        GCS_BUCKET : content[GCS_BUCKET].strip()}


def _get_post_job_configs_error(content):
    """Checks that the input content has the required fields.

       Returns:
           An error response dictionary if there is an error with the input
           content; or None if there is no error.
    """
    for field in POST_CONFIG_FIELDS:
        if field not in content:
            return _MISSING_FIELD_ERROR
    if not (_UNIX_SOURCE_PATH_PATTERN.match(content[FILE_SYSTEM_DIR]) or
        _WINDOWS_SOURCE_PATH_PATTERN.match(content[FILE_SYSTEM_DIR])):
        return _PATH_PATTERN_ERROR
    if not _CONFIG_ID_PATTERN.match(content[CONFIG_ID]):
        return _CONFIG_FORMAT_ERROR
    if not _BUCKET_PATTERN.match(content[GCS_BUCKET]):
        return _BUCKET_PATTERN_ERRROR
    return None

def _get_delete_job_configs_error(content):
    """Checks that the input content is in the correct format.

       Returns:
           An error response dictionary if there is an error with the input
           content, or None if there is no error.
    """
    formatting_error_response = {
        'error' : 'Request is incorrectly formatted.',
        'message' : 'The request should be a list of job config id strings.'
    }
    if not isinstance(content, list):
        return formatting_error_response
    if len(content) == 0:
        return formatting_error_response
    for item in content:
        if not (isinstance(item, str) or isinstance(item, unicode)):
            return formatting_error_response
    return None

def _get_error_for_get_tasks_of_status(last_modified, task_status):
    """Checks that the input for the get tasks of status route is in the
       correct format.

       Returns:
           An error response dictionary if there is an error with the input, or
           None if there is no error.
    """
    current_time = util.get_unix_nano()
    if last_modified is not None and (
        last_modified > current_time or last_modified < 0):
        return _INVALID_TIMESTAMP_ERROR
    if task_status not in TaskStatus.Type.values():
        return _INVALID_TASK_STATUS_ERROR
    return None

def _get_tasks_failure_error(last_modified, failure_type):
    """Checks that the input for the get tasks of failure type route is in the
       correct format.

       Returns:
           An error response dictionary if there is an error with the input, or
           None if there is no error.
    """
    current_time = util.get_unix_nano()
    if last_modified is not None and (
        last_modified > current_time or last_modified < 0):
        return _INVALID_TIMESTAMP_ERROR
    if failure_type not in TaskFailureType.Type.values():
        return _INVALID_TASK_FAILURE_TYPE_ERROR
    return None

@APP.route('/projects/<project_id>/jobconfigs/delete',
           methods=['OPTIONS', 'POST'])
@crossdomain(origin=APP.config['CLIENT'], headers=_ALLOWED_HEADERS)
def delete_job_configs(project_id):
    """Handles a request to delete job configurations.
    """
    spanner_wrapper = SpannerWrapper(_get_credentials(),
                                     project_id,
                                     APP.config['SPANNER_INSTANCE'],
                                     APP.config['SPANNER_DATABASE'])
    if request.method == 'POST':
        content = request.json
        error = _get_delete_job_configs_error(content)
        if error:
            return jsonify(error), httplib.BAD_REQUEST
        result = spanner_wrapper.delete_job_configs(content)
        if len(result['indelible_configs']) > 0:
            error = {
                'error': 'Error deleting jobs',
                'message': ('Could not delete the following jobs because '
                            'they have tasks in progress: {0}.')
                            .format(', '.join(result['indelible_configs']))
            }
            return jsonify(error), httplib.BAD_REQUEST

        return jsonify(result['delible_configs']), httplib.OK


@APP.route('/projects/<project_id>/jobconfigs',
           methods=['GET', 'OPTIONS', 'POST'])
@crossdomain(origin=APP.config['CLIENT'], headers=_ALLOWED_HEADERS)
def job_configs(project_id):
    """Handles geting a list of job configs and creating a new job
       configuration
    """
    spanner_wrapper = SpannerWrapper(_get_credentials(),
                                     project_id,
                                     APP.config['SPANNER_INSTANCE'],
                                     APP.config['SPANNER_DATABASE'])
    if not _PROJECT_ID_PATTERN.match(project_id):
        return jsonify(_PROJECT_ID_FORMAT_ERROR), httplib.BAD_REQUEST
    if request.method == 'GET':
        configs = spanner_wrapper.get_job_configs()
        return jsonify(configs)
    elif request.method == 'POST':
        content = request.json
        error = _get_post_job_configs_error(content)
        if error:
            return jsonify(error), httplib.BAD_REQUEST
        job_spec = _get_post_job_configs_job_spec(content)
        spanner_wrapper.create_new_job(content[CONFIG_ID], job_spec)
        return jsonify({}), httplib.OK

@APP.route('/projects/<project_id>/jobrun/<config_id>',
           methods=['GET', 'OPTIONS'])
@crossdomain(origin=APP.config['CLIENT'], headers=_ALLOWED_HEADERS)
def get_job_run(project_id, config_id):
    """Gets the job run info for the input config id.

    Args:
        project_id: The id of the project.
        config_id: The id of the job config.

    Returns:
        On success-
            200, A JSON object containining the job run info with the
                 job config info.
        On failure-
            400, Bad request if the job config id is not valid or the project
                 id is not valid.
            404, Not found if a job run with the given ids does not exist.
    """
    spanner_wrapper = SpannerWrapper(_get_credentials(),
                                     project_id,
                                     APP.config['SPANNER_INSTANCE'],
                                     APP.config['SPANNER_DATABASE'])
    if not _PROJECT_ID_PATTERN.match(project_id):
        return jsonify(_PROJECT_ID_FORMAT_ERROR), httplib.BAD_REQUEST
    if not _CONFIG_ID_PATTERN.match(config_id):
        return jsonify(_CONFIG_FORMAT_ERROR), httplib.BAD_REQUEST
    result = spanner_wrapper.get_job_run(config_id)
    if not result:
        return jsonify(_CONFIG_NOT_FOUND_ERROR), httplib.NOT_FOUND
    return jsonify(result), httplib.OK

@APP.route(
'/projects/<project_id>/tasks/<config_id>/status/<task_status>',
methods=['OPTIONS', 'GET'])
@crossdomain(origin=APP.config['CLIENT'], headers=_ALLOWED_HEADERS)
def get_tasks_of_status(project_id, config_id, task_status):
    """Handles GET requests for tasks of a specified status.
    This route has two query parameters:
        lastModifiedBefore- Only return tasks modified before this timestamp.
    Args:
        project_id: The id of the project.
        config_id: The id of the job config for the desired tasks
        task_status: The task status code for the tasks.
    Returns:
        The tasks of the input status that are found in the database. The
        maximum number of tasks returned is equal to the _NUM_OF_TASKS constant
        in the spannerwrapper.py file.
    """
    spanner_wrapper = SpannerWrapper(_get_credentials(),
                                     project_id,
                                     APP.config['SPANNER_INSTANCE'],
                                     APP.config['SPANNER_DATABASE'])
    if not _PROJECT_ID_PATTERN.match(project_id):
        return jsonify(_PROJECT_ID_FORMAT_ERROR), httplib.BAD_REQUEST
    if not _CONFIG_ID_PATTERN.match(config_id):
        return jsonify(_CONFIG_FORMAT_ERROR), httplib.BAD_REQUEST
    last_modified_before = _get_int_param(request, 'lastModifiedBefore')
    task_status_int = int(task_status)
    error = _get_error_for_get_tasks_of_status(last_modified_before,
        task_status_int)
    if error:
        return jsonify(error), httplib.BAD_REQUEST

    return jsonify(spanner_wrapper.get_tasks_of_status(
        config_id,
        task_status_int,
        last_modified_before
    ))

@APP.route(
'/projects/<project_id>/tasks/<config_id>/failuretype/<failure_type>',
methods=['OPTIONS', 'GET'])
@crossdomain(origin=APP.config['CLIENT'], headers=_ALLOWED_HEADERS)
def get_tasks_of_failure_type(project_id, config_id, failure_type):
    """Handles GET requests for tasks of a specified status.
    This route has one query parameter:
        lastModifiedBefore- Only return tasks modified before this timestamp.
    Args:
        project_id: The id of the project.
        config_id: The id of the job config for the desired tasks
        failure_type: The failure type code of the tasks
    Returns:
        The tasks of the input status that are found in the database. The
        maximum number of tasks returned is equal to the _NUM_OF_TASKS constant
        in the spannerwrapper.py file.
    """
    spanner_wrapper = SpannerWrapper(_get_credentials(),
                                     project_id,
                                     APP.config['SPANNER_INSTANCE'],
                                     APP.config['SPANNER_DATABASE'])
    if not _PROJECT_ID_PATTERN.match(project_id):
        return jsonify(_PROJECT_ID_FORMAT_ERROR), httplib.BAD_REQUEST
    if not _CONFIG_ID_PATTERN.match(config_id):
        return jsonify(_CONFIG_FORMAT_ERROR), httplib.BAD_REQUEST
    last_modified_before = _get_int_param(request, 'lastModifiedBefore')
    failure_type_int = int(failure_type)
    error = _get_tasks_failure_error(last_modified_before,
        failure_type_int)
    if error:
        return jsonify(error), httplib.BAD_REQUEST

    return jsonify(spanner_wrapper.get_tasks_of_failure_type(
        config_id,
        failure_type_int,
        last_modified_before
    ))

@APP.route('/projects/<project_id>/infrastructure-status',
           methods=['OPTIONS', 'GET'])
@crossdomain(origin=APP.config['CLIENT'], headers=_ALLOWED_HEADERS)
def infrastructure_status(project_id):
    """Gets the ingest infrastructure status.

    Responds with a JSON object contains all the infrastructure component
    statuses. Each status is a string from one of the following values
    ('RUNNING', 'DEPLOYING', 'DELETING', 'FAILED', 'NOT_FOUND', or 'UNKNOWN')
    """
    # TODO(b/65586429): Getting the infrastructure status API may take the
    # resources (in the request body) to query for.
    return jsonify(infra_util.infrastructure_status(_get_credentials(),
                                                    project_id))

@APP.route('/projects/<project_id>/create-infrastructure',
           methods=['OPTIONS', 'POST'])
@crossdomain(origin=APP.config['CLIENT'], headers=_ALLOWED_HEADERS)
def create_infrastructure(project_id):
    """Creates the ingest infrastructure.
    """
    # TODO(b/65754348): Creating the infrastructure API may take the resources
    # (in the request body) to create.
    dcp_docker_image = (None if APP.config.get('SKIP_RUNNING_DCP', False) else
                        APP.config['DCP_DOCKER_IMAGE'])

    if not dcp_docker_image and not APP.config['DEBUG']:
        logging.critical('DCP docker image must be specified in prod '
                         'environment.')
        raise ServerError('Internal Server Error')

    infra_util.create_infrastructure(
        _get_credentials(), project_id, dcp_docker_image)
    return jsonify({})

@APP.route('/projects/<project_id>/tear-down-infrastructure',
           methods=['OPTIONS', 'POST'])
@crossdomain(origin=APP.config['CLIENT'], headers=_ALLOWED_HEADERS)
def tear_infrastructure(project_id):
    """Tears down the ingest infrastructure.
    """
    # TODO(b/65754348): Tearing the infrastructure API may take the resources
    # (in the request body) to tear down.
    infra_util.tear_infrastructure(_get_credentials(), project_id)
    return jsonify({})

def _get_int_param(get_request, param_name):
    """Returns the int GET parameter named param_name from the given request.

    Returns the int value of param_name in the given GET request.
    If the given parameter is not set in the request, returns None.

    Args:
        get_request: The GET request
        param_name: The name of the GET parameter

    Returns:
        The value of the parameter, or None if it is not set.

    Raises:
        BadRequest with a helpful message if the type of the value
        is something other than int.
    """
    try:
        value = get_request.args.get(param_name)
        if value is not None:
            return int(get_request.args.get(param_name))
    except ValueError:
        raise BadRequest("GET param '%s' must be a valid integer. "
                         "Current value: %s" % (param_name, value))

@APP.errorhandler(httplib.INTERNAL_SERVER_ERROR)
@crossdomain(origin=APP.config['CLIENT'], headers=_ALLOWED_HEADERS,
             methods=['GET', 'OPTIONS', 'POST'])
def server_error(error):
    """Handles any 500 (Internal Server Error) response.

    This function will be passed any uncaught exceptions in addition to
    explicit 500 errors. This handler is not used in debug mode."""
    traceback_string = traceback.format_exc()
    logging.error(('A request could not be completed due to an error: %s.\n'
                   'Request path: %s\n'
                   'Request args: %s\n'
                   'Request json: %s\n'
                   '%s'),
                   str(error), str(request.path),
                   json.dumps(request.args),
                   json.dumps(request.json), traceback_string)
    response = {
        'error': 'Internal Server Error',
        'message': str(error),
        'traceback': traceback_string
    }
    return jsonify(response), httplib.INTERNAL_SERVER_ERROR

@APP.errorhandler(Conflict)
@APP.errorhandler(Forbidden)
@APP.errorhandler(NotFound)
@APP.errorhandler(PreconditionFailed)
@APP.errorhandler(Unauthorized)
@APP.errorhandler(BadRequest)
@crossdomain(origin=APP.config['CLIENT'], headers=_ALLOWED_HEADERS,
             methods=['GET', 'OPTIONS', 'POST'])
def google_exception_handler(error):
    """Handles any google exception errors."""
    traceback_string = traceback.format_exc()
    logging.error(('A request resulted in a error code %d with message: %s.\n'
                   'Request path: %s\n'
                   'Request args: %s\n'
                   'Request json: %s\n'
                   ' %s'),
                   error.code, str(error), str(request.path),
                   json.dumps(request.args), json.dumps(request.json),
                   traceback_string)
    response = {
        'error': httplib.responses[error.code],
        'message': str(error),
        'traceback': traceback_string
    }
    return jsonify(response), error.code

@APP.errorhandler(httplib.NOT_FOUND)
@crossdomain(origin=APP.config['CLIENT'], headers=_ALLOWED_HEADERS,
             methods=['GET', 'OPTIONS', 'POST'])
def bad_url(error):
    """ Handles 404 Not Found errors caused by a bad request url"""
    # pylint: disable=unused-argument
    response = {
        'error': 'NotFound',
        'message': 'The requested url could not be found on the server.'
    }
    return jsonify(response), httplib.NOT_FOUND

def main():
    """Executes the main logic when the function is run from the command line.
    Used when executed on a local workstation.
    """
    parser = argparse.ArgumentParser(
        description='Cloud ingest local backend server')

    parser.add_argument('--skip-running-dcp', '-sdcp', action='store_true',
                        help='Skip running the DCP when creating an '
                             'infra-structure.',
                        default=False)
    args = parser.parse_args()
    APP.config['SKIP_RUNNING_DCP'] = args.skip_running_dcp

    APP.run(
        host=APP.config['HOST'],
        port=APP.config['PORT'],
        debug=APP.config['DEBUG'])


if __name__ == '__main__':
    main()
