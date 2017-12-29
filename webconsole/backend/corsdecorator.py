"""A decorator for allowing cross-site HTTP requests.

Source: http://flask.pocoo.org/snippets/56/

"""
from datetime import timedelta
from functools import update_wrapper
from flask import current_app
from flask import make_response
from flask import request


def crossdomain(origin=None, methods=None, headers=None,
                max_age=21600, attach_to_all=True,
                automatic_options=True):
    """Return a CORS decorator."""
    if methods is not None:
        methods = ', '.join(sorted(x.upper() for x in methods))
    if headers is not None and not isinstance(headers, basestring):
        headers = ', '.join(x.upper() for x in headers)
    if not isinstance(origin, basestring):
        origin = ', '.join(origin)
    if isinstance(max_age, timedelta):
        # pylint: disable=maybe-no-member
        max_age = max_age.total_seconds()

    def get_methods():
        # pylint: disable=missing-docstring
        if methods is not None:
            return methods

        options_resp = current_app.make_default_options_response()
        return options_resp.headers['allow']

    def decorator(flask_view):
        # pylint: disable=missing-docstring
        def wrapped_function(*args, **kwargs):
            # pylint: disable=missing-docstring
            if automatic_options and request.method == 'OPTIONS':
                resp = current_app.make_default_options_response()
            else:
                resp = make_response(flask_view(*args, **kwargs))
            if not attach_to_all and request.method != 'OPTIONS':
                return resp

            header_map = resp.headers

            header_map['Access-Control-Allow-Origin'] = origin
            header_map['Access-Control-Allow-Methods'] = get_methods()
            header_map['Access-Control-Max-Age'] = str(max_age)
            if headers is not None:
                header_map['Access-Control-Allow-Headers'] = headers
            return resp

        flask_view.provide_automatic_options = False
        return update_wrapper(wrapped_function, flask_view)
    return decorator
