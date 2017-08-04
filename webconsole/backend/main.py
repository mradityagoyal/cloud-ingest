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
            return jsonify({}), httplib.BAD_REQUEST

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
        response = jsonify(SPANNER_CLIENT.get_job_runs())
        return response
    elif request.method == 'POST':
        content = request.json
        if 'JobConfigId' not in content or 'JobRunId' not in content:
            response = {
                'error':
                'missing required property',
                'message': ('Missing at least one of the required properties: '
                            '[\'JobConfigId\', \'JobRunId\']')
            }
            return jsonify({}), httplib.BAD_REQUEST

        created = SPANNER_CLIENT.create_job_run(content['JobConfigId'],
                                                content['JobRunId'])
        if created:
            created_job_run = SPANNER_CLIENT.get_job_run(
                content['JobConfigId'], content['JobRunId'])
            return jsonify(created_job_run), httplib.CREATED
        # TODO(b/64075962) Better error handling
        return jsonify({}), httplib.BAD_REQUEST

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


if __name__ == '__main__':
    # Used when running locally
    APP.run(
        host=APP.config['HOST'],
        port=APP.config['PORT'],
        debug=APP.config['DEBUG'])
