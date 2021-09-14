from common.auth import *
from common.tools import *

# handle post requests for specified 'url'. 
# 'name' parameter denotes the display name in the statistics printed by locust
# 
# return either empty string if not successful or resource 'id'
def handle_post(self, url, payload, name, full_json=False):
  with self.client.post(url, json=payload, verify=False, catch_response=True, name=name) as response:
    if response.status_code == 409:
      response.success() # ignore unlike 409 errors when generated resource id is duplicated
    if response.status_code == 401:
      response.success()
      get_token(self)
    if response.status_code == 429: # limit exceeded
      return '429'
    if response.status_code == 202:
      try:
        response_json = response.json()
        if 'id' in response_json:
          if full_json == True:
            return response_json
          else:
            return response_json['id']
      except ValueError: # no json response
        return ''
    return ''

# handle delete requests for specified 'url'. 
# 'name' parameter denotes the display name in the statistics printed by locust
#
# return either 0 (meaning no success in deletion) or 204 if the deletion was successful
def handle_delete_by_id(self, url, name):
  with self.client.delete(url, verify=False, catch_response=True, name=name) as response:
    if response.status_code == 401:
      response.success()
      get_token(self)
    if response.status_code >= 500 or response.status_code == 404:
      try:
        response_output = response.json()
        if 'reason' in response_output:
          reason = response_output['reason']
          # due to concurrency sometimes resources have already been deleted
          if '404' in reason or 'failed to delete service account' in reason or 'Unable to find DinosaurResource' in reason or 'not found' in reason:
            response.success()
            return 202
      except ValueError: # no json response
        return 500
    if response.status_code < 300: # success
      return response.status_code
    return 500

# handle get requests for specified 'url'. 
# 'name' parameter denotes the display name in the statistics printed by locust.
# 'return_json_response' set to True will return json response
def handle_get(self, url, name, return_json_response=False):
  with self.client.get(url, verify=False, catch_response=True, name=name) as response:
    if response.status_code == 401:
      response.success()
      get_token(self)
    if response.status_code == 404: # 404 isn't a failing request
      response.success()
    if return_json_response == True:
      try:
        response_json = response.json()
        return response_json
      except ValueError: # no json response
        return []
