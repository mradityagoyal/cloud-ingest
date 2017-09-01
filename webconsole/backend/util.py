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
