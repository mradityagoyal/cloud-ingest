"""Contains the utility functions for the webconsole backend.
"""

import time

def dict_values_are_recursively(dictionary, desired_value):
    """Checks whether all the values of a dictionary are the same as the
    desired_value. The dictionary values may contain arbitrary objects or
    another dictionaries. If a value is a dictionary (D), the method recursively
    checks all the values of that dictionary (D) is equal to the desired_value.

    Args:
        dictionary: A dictionary to check.
        desired_value: The value to check against.

    Returns:
        Whether all the values of the dictionary (recursively) are equals to the
        desired_value.
    """
    for value in dictionary.itervalues():
        if isinstance(value, dict):
            if not dict_values_are_recursively(value, desired_value):
                return False
        else:
            if value != desired_value:
                return False
    return True

def dict_has_values_recursively(dictionary, desired_values):
    """Checks whether any of the dictionary values are the same as the any of
    the desired_values. The dictionary values may contain arbitrary objects or
    another dictionaries. If a value is a dictionary (D), the method recursively
    checks all the values of that dictionary (D) has any of the desired_values.

    Args:
        dictionary: A dictionary to check.
        desired_values: Set of the values to check against.

    Returns:
        Whether any of the dictionary values (recursively) are equals to any
        value in the desired_values set.
    """
    for value in dictionary.itervalues():
        if isinstance(value, dict):
            if dict_has_values_recursively(value, desired_values):
                return True
        else:
            if value in desired_values:
                return True
    return False

def get_unix_nano():
    """Returns the current Unix time in nanoseconds

    Returns:
        An integer representing the current Unix time in nanoseconds
    """
    # time.time() returns Unix time in seconds. Multiply by 1e9 to get
    # the time in nanoseconds
    return int(time.time() * 1e9)
