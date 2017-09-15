"""Contains the utility functions for the webconsole backend.
"""

import json

def json_to_dictionary_in_field(dictionary_list, json_field):
    """Reads the json in the input dictionary fields for each dictionary in the
       dictionary list. Converts json into a dictionary. It only looks at one
       level deep.

          Args:
            dictionary_list: A list of dictionaries.
            json_field: The field to read as json and convert to
                a dictionary.

          Returns:
            A list of dictionaries with the input field converted from json to a
            dictionary.

    """
    result_list = []
    for dictionary in dictionary_list:
        dictionary[json_field] = json.loads(
            dictionary[json_field])
        result_list.append(dictionary)
    return result_list

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
