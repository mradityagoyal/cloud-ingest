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
# pylint: disable=import-error,no-name-in-module
import httplib
import logging
from corsdecorator import crossdomain  # To allow requests from the front-end
from flask import Flask
from flask import jsonify
from flask import request
from spannerwrapper import SpannerWrapper

APP = Flask(__name__)
APP.config.from_pyfile('ingestwebconsole.default_settings')
APP.config.from_envvar('INGEST_CONFIG_PATH')

SPANNER_CLIENT = SpannerWrapper(APP.config['JSON_KEY_PATH'],
                                APP.config['SPANNER_INSTANCE'],
                                APP.config['SPANNER_DATABASE'])
_DEFAULT_PAGE_SIZE = 25

# Allowed headers in cross-site http requests.
_ALLOWED_HEADERS = ['Content-Type', 'Authorization']

@APP.route('/jobconfigs', methods=['GET', 'OPTIONS', 'POST'])
@crossdomain(origin=APP.config['CLIENT'], headers=_ALLOWED_HEADERS)
def job_configs():
    """Handle all job config related requests."""
    if request.method == 'GET':
        response = jsonify(SPANNER_CLIENT.get_job_configs())
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

        created = SPANNER_CLIENT.create_job_config(content['JobConfigId'],
                                                   content['JobSpec'])
        if created:
            created_config = SPANNER_CLIENT.get_job_config(
                content['JobConfigId'])
            return jsonify(created_config), httplib.CREATED
        # TODO(b/64075962) Better error handling
        return jsonify({}), httplib.BAD_REQUEST


@APP.route('/jobruns', methods=['GET', 'OPTIONS', 'POST'])
@crossdomain(origin=APP.config['CLIENT'], headers=_ALLOWED_HEADERS)
def job_runs():
    """Handle all job run related requests."""
    if request.method == 'GET':
        created_before = _get_int_param(request, 'createdBefore')
        num_runs = _get_int_param(request, 'pageSize') or _DEFAULT_PAGE_SIZE

        return jsonify(SPANNER_CLIENT.get_job_runs(
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

        created = SPANNER_CLIENT.create_job_run(content['JobConfigId'],
                                                content['JobRunId'])
        if created:
            created_job_run = SPANNER_CLIENT.get_job_run(
                content['JobConfigId'], content['JobRunId'])
            return jsonify(created_job_run), httplib.CREATED
        # TODO(b/64075962) Better error handling
        return jsonify({}), httplib.BAD_REQUEST

@APP.route('/tasks/<config_id>/<run_id>', methods=['GET'])
@crossdomain(origin=APP.config['CLIENT'], headers=_ALLOWED_HEADERS)
def tasks(config_id, run_id):
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
    last_modified_before = _get_int_param(request, 'lastModifiedBefore')
    task_type = request.args.get('type')
    num_tasks = _get_int_param(request, 'pageSize') or _DEFAULT_PAGE_SIZE

    return jsonify(SPANNER_CLIENT.get_tasks_for_run(
        config_id,
        run_id,
        num_tasks,
        last_modified=last_modified_before,
        task_type=task_type
    ))

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
def value_error(error):
    """Handles any uncaught value errors."""
    logging.info('A bad request was made: %s', str(error))
    response = {
        'error': 'Bad Request',
        'message': str(error)
    }
    return jsonify(response), httplib.BAD_REQUEST


if __name__ == '__main__':
    # Used when running locally
    APP.run(
        host=APP.config['HOST'],
        port=APP.config['PORT'],
        debug=APP.config['DEBUG'])
