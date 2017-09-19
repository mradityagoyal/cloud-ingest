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
import httplib
import logging
import re
from corsdecorator import crossdomain  # To allow requests from the front-end
from flask import Flask
from flask import jsonify
from flask import request
from google.cloud.exceptions import Conflict
from google.cloud.exceptions import NotFound
from google.cloud.exceptions import Forbidden
from google.cloud.exceptions import PreconditionFailed
from google.cloud.exceptions import Unauthorized
from google.oauth2.credentials import Credentials

import infra_util
from spannerwrapper import SpannerWrapper
from create_infra.constants import DCP_INSTANCE_DOCKER_IMAGE


APP = Flask(__name__)
APP.config.from_pyfile('ingestwebconsole.default_settings')
APP.config.from_envvar('INGEST_CONFIG_PATH')

_DEFAULT_PAGE_SIZE = 25

# Allowed headers in cross-site http requests.
_ALLOWED_HEADERS = ['Content-Type', 'Authorization']

_AUTH_HEADER_REGEX = re.compile(r'^\s*Bearer\s+(?P<access_token>[^\s]*)$')

# TODO(b/65846311): Temporary constants used to create the job run and first
# list task when a job config is created. Eventually DCP should manage
# scheduling/creating job runs and first listing tasks.
_FIRST_JOB_RUN_ID = "first-job-run"
_LIST_TASK_ID = "list"

def _get_credentials():
    """Gets OAuth2 credentials from request Authorization headers."""
    auth_header = request.headers.get('Authorization')
    if not auth_header:
        raise Unauthorized(('Missing Authorization header. Please add ' +
                            'Authorization header in the format "Bearer ' +
                            '<access_token>".'))
    match = _AUTH_HEADER_REGEX.match(auth_header)
    if not match:
        raise ValueError(
            'Invalid Authorization header format, the Authorization header '
            'should be in the format "Bearer <access_token>".')
    access_token = match.group('access_token')

    credentials = Credentials(access_token)
    return credentials


@APP.route('/projects/<project_id>/jobconfigs',
           methods=['GET', 'OPTIONS', 'POST'])
@crossdomain(origin=APP.config['CLIENT'], headers=_ALLOWED_HEADERS)
def job_configs(project_id):
    """Handle all job config related requests."""
    spanner_wrapper = SpannerWrapper(_get_credentials(),
                                     project_id,
                                     APP.config['SPANNER_INSTANCE'],
                                     APP.config['SPANNER_DATABASE'])
    if request.method == 'GET':
        response = jsonify(spanner_wrapper.get_job_configs())
        return response
    elif request.method == 'POST':
        content = request.json
        if 'JobConfigId' not in content or 'JobSpec' not in content:
            response = {
                'error':
                'missing required property',
                'message': ('Missing at least one of the required properties: '
                            '[\'JobConfigId\', \'JobSpec\']')
            }
            return jsonify(response), httplib.BAD_REQUEST

        spanner_wrapper.create_job_config(content['JobConfigId'],
                                          content['JobSpec'])

        # TODO(b/65943019): The web console should not schedule the job runs and
        # should not create the first task. Remove that after the functionality
        # is added to the DCP.
        spanner_wrapper.create_job_run(
            content['JobConfigId'], _FIRST_JOB_RUN_ID, initial_total_tasks=1)
        spanner_wrapper.create_job_run_first_list_task(
            content['JobConfigId'], _FIRST_JOB_RUN_ID, _LIST_TASK_ID,
            content['JobSpec'])

        created_config = spanner_wrapper.get_job_config(
            content['JobConfigId'])
        return jsonify(created_config), httplib.CREATED


@APP.route('/projects/<project_id>/jobruns', methods=['GET', 'OPTIONS', 'POST'])
@crossdomain(origin=APP.config['CLIENT'], headers=_ALLOWED_HEADERS)
def job_runs(project_id):
    """Handle all job run related requests."""
    spanner_wrapper = SpannerWrapper(_get_credentials(),
                                     project_id,
                                     APP.config['SPANNER_INSTANCE'],
                                     APP.config['SPANNER_DATABASE'])
    if request.method == 'GET':
        created_before = _get_int_param(request, 'createdBefore')
        num_runs = _get_int_param(request, 'pageSize') or _DEFAULT_PAGE_SIZE

        return jsonify(spanner_wrapper.get_job_runs(
            num_runs,
            created_before=created_before
        ))
    elif request.method == 'POST':
        content = request.json
        if 'JobConfigId' not in content or 'JobRunId' not in content:
            response = {
                'error':
                'missing required property',
                'message': ('Missing at least one of the required properties: '
                            '[\'JobConfigId\', \'JobRunId\']')
            }
            return jsonify(response), httplib.BAD_REQUEST

        spanner_wrapper.create_job_run(content['JobConfigId'],
                                       content['JobRunId'])
        created_job_run = spanner_wrapper.get_job_run(
            content['JobConfigId'], content['JobRunId'])
        return jsonify(created_job_run), httplib.CREATED

@APP.route('/projects/<project_id>/jobruns/<config_id>/<run_id>',
           methods=['GET', 'OPTIONS'])
@crossdomain(origin=APP.config['CLIENT'], headers=_ALLOWED_HEADERS)
def job_run(project_id, config_id, run_id):
    """Responds with the specified job run, or 404 Not Found.

    Args:
        project_id: The id of the project.
        config_id: The id of the job config
        run_id: The id of the job run

    Returns:
        On success-
            200, A JSON job run object
        On failure-
            404, Not found if a job run with the given ids does not exist
    """
    spanner_wrapper = SpannerWrapper(_get_credentials(),
                                     project_id,
                                     APP.config['SPANNER_INSTANCE'],
                                     APP.config['SPANNER_DATABASE'])
    result = spanner_wrapper.get_job_run(config_id, run_id)
    if not result:
        raise NotFound(("Could not find a job run with config_id: %s,"
                        " run_id: %s") % (config_id, run_id))
    return jsonify(result), httplib.OK

@APP.route('/projects/<project_id>/tasks/<config_id>/<run_id>',
           methods=['OPTIONS', 'GET'])
@crossdomain(origin=APP.config['CLIENT'], headers=_ALLOWED_HEADERS)
def tasks(project_id, config_id, run_id):
    """Handles GET requests for tasks.

    This route has several optional query parameters.
        pageSize- The number of tasks to return. Default is _DEFAULT_PAGE_SIZE.
                  Values less than 1 and greater than 10,000 result in a
                  response of 400 BAD_REQUEST.
        lastModifiedBefore- The unix epoch time used to filter tasks. Only tasks
                            with last modified times before the given time
                            will be returned.
        type- Only tasks with the given type will be returned.

    Args:
        project_id: The id of the project.
        config_id: The id of the job config for the desired tasks
        run_id: The id of the job run for the desired tasks

    Returns:
        On success-
            200, A JSON list of pageSize (defaults to _DEFAULT_PAGE_SIZE)
                 matching tasks
        On failure-
            400, Bad request due to invalid values for query params
            500, Any uncaught exception is raised during the processing of
                 the request
    """
    spanner_wrapper = SpannerWrapper(_get_credentials(),
                                     project_id,
                                     APP.config['SPANNER_INSTANCE'],
                                     APP.config['SPANNER_DATABASE'])
    last_modified_before = _get_int_param(request, 'lastModifiedBefore')
    task_type = request.args.get('type')
    num_tasks = _get_int_param(request, 'pageSize') or _DEFAULT_PAGE_SIZE

    return jsonify(spanner_wrapper.get_tasks_for_run(
        config_id,
        run_id,
        num_tasks,
        last_modified=last_modified_before,
        task_type=task_type
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
    dcp_docker_image = APP.config.get('DCP_DOCKER_IMAGE',
                                      DCP_INSTANCE_DOCKER_IMAGE)
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
        ValueError with a helpful message if the type of the value
        is something other than int.
    """
    try:
        value = get_request.args.get(param_name)
        if value is not None:
            return int(get_request.args.get(param_name))
    except ValueError:
        raise ValueError("GET param '%s' must be a valid integer. "
                         "Current value: %s" % (param_name, value))

@APP.errorhandler(httplib.INTERNAL_SERVER_ERROR)
@crossdomain(origin=APP.config['CLIENT'], headers=_ALLOWED_HEADERS,
             methods=['GET', 'OPTIONS', 'POST'])
def server_error(error):
    """Handles any 500 (Internal Server Error) response.

    This function will be passed any uncaught exceptions in addition to
    explicit 500 errors. This handler is not used in debug mode."""
    logging.error('A request could not be completed due to an error: %s',
                  str(error))
    response = {
        'error': 'Internal Server Error',
        'message': ('An internal server error occurred and '
                    'the request could not be completed')
    }
    return jsonify(response), httplib.INTERNAL_SERVER_ERROR

@APP.errorhandler(ValueError)
@crossdomain(origin=APP.config['CLIENT'], headers=_ALLOWED_HEADERS,
             methods=['GET', 'OPTIONS', 'POST'])
def value_error(error):
    """Handles any uncaught value errors."""
    logging.info('A bad request was made: %s', str(error))
    response = {
        'error': 'Bad Request',
        'message': str(error)
    }
    return jsonify(response), httplib.BAD_REQUEST

@APP.errorhandler(Conflict)
@APP.errorhandler(Forbidden)
@APP.errorhandler(NotFound)
@APP.errorhandler(PreconditionFailed)
@APP.errorhandler(Unauthorized)
@crossdomain(origin=APP.config['CLIENT'], headers=_ALLOWED_HEADERS,
             methods=['GET', 'OPTIONS', 'POST'])
def google_exception_handler(error):
    """Handles any google exception errors."""
    logging.info('A request resulted in a error code %d with message: %s',
                 error.code, str(error))
    response = {
        'error': httplib.responses[error.code],
        'message': str(error.message)
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


if __name__ == '__main__':
    # Used when running locally
    APP.run(
        host=APP.config['HOST'],
        port=APP.config['PORT'],
        debug=APP.config['DEBUG'])
