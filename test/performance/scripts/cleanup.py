import os, requests, subprocess, sys, time, urllib3

# disable ssl check warnings
urllib3.disable_warnings()

# read env vars
api_host = os.getenv('API_HOST')
file_path = os.getenv('FILE_PATH')
resource = os.getenv('RESOURCE')
if str(api_host) == 'None' or str(file_path) == 'None' or str(resource) == 'None':
  sys.exit('Some of required params not specified (API_HOST, FILE_PATH or RESOURCE)')

# get short living token and return request headers
def get_headers():
  get_token = subprocess.check_output("ocm token", shell=True)
  token = get_token.decode('utf-8').rstrip('\n')
  headers = {'Content-Type': 'application/json', 'Authorization': 'Bearer ' + token}
  return headers

# read file with resources IDs
lines = open(file_path).read().splitlines()

# get headers initially
headers = get_headers()

# set counter for deleting resources and iterate over those resources' ids
i = 0
while i < len(lines):
  # set utl
  url = f'{api_host}/api/managed-services-api/v1/{resource}/{lines[i]}?async=true'
  r = requests.delete(url, headers=headers, verify=False)
  print(f'{resource} deletion -> id: {lines[i]} -> status code: {str(r.status_code)}')
  if r.status_code <= 204 or r.status_code == 404: # 404 or 202 are success states
    i = i + 1
  if r.status_code == 401: # if token expired - get new one
    headers = get_headers()
  # small timeout between requests
  time.sleep(0.1)
