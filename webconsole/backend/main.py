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

The application is set-up to be a RESTful API.
"""
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


@APP.route('/jobconfigs', methods=['GET', 'OPTIONS', 'POST'])
@crossdomain(origin=APP.config['CLIENT'], headers=['Content-Type'])
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
@crossdomain(origin=APP.config['CLIENT'], headers=['Content-Type'])
def job_runs():
    """Handle all job run related requests."""
    if request.method == 'GET':
        num_runs = get_int_param(request, 'pageSize')
        created_before = get_int_param(request, 'createdBefore')

        result = None
        if num_runs is None:
            result = SPANNER_CLIENT.get_job_runs(created_before=created_before)
        else:
            result = SPANNER_CLIENT.get_job_runs(max_num_runs=num_runs,
                                                 created_before=created_before)
        return jsonify(result)
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

def get_int_param(get_request, param_name):
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
