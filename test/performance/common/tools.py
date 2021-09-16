import json, random, requests, string, subprocess, sys, time

# golang API helper urls used to persist config
dinosaur_id_write_url = 'http://api:8099/write_dinosaur_id'

# check if container id of the container making the request was elected
# to create dinosaur requests
def check_dinosaur_create_enabled():
  get_container_id_output = subprocess.check_output("cat /proc/self/cgroup | head -n 1 | cut -d '/' -f3", shell=True)
  container_id = get_container_id_output.decode('utf-8')
  container_info = {
                     'containerId': container_id,
                   }
  headers = {'content-type': 'application/json'}
  container_id_sent = False
  while container_id_sent == False:
    r = requests.post('http://api:8099/dinosaur_create_container_id', data=json.dumps(container_info), headers=headers)
    # retry persisting config if unsuccessful
    if r.status_code != 200:
      time.sleep(random.uniform(1, 2))
      continue
    else: 
      container_id_sent = True
  return r.text == container_id

# get_ids_from_list iterates over passed list and returns ids found among the passed list items
def get_ids_from_list(list):
  ids = []
  for item in list:
    if 'id' in item:
      ids.append(item['id'])
  return ids

# returns items, if in data structure, otherwise return an empty array
def get_items_from_json_response(json_response):
  if 'items' in json_response:
    return json_response['items']
  
  return []

# random_string is used to create dinosaur_request names
def random_string(length):
  return ''.join(random.choice(string.ascii_lowercase) for i in range(length))

# random_status - get random status of dinosaur_cluster
def random_status():
  return random.choice(['accepted', 'preparing', 'provisioning', 'ready'])

# dinosaur_json is used to create json payload 
def dinosaur_json():
  return {
           'name': random_string(15)
         }

# get_random_search query to get with search param
def get_random_search():
  return f'name <> {random_string(15)}'

# get random id from list
def get_random_id(list):
  return random.choice(list)

def safe_delete(list, item):
  if item in list:
    list.remove(item)

def persist_dinosaur_id(dinosaur_id):
  config = {
             'dinosaurId': dinosaur_id,
           }
  persist_to_api_helper(dinosaur_id_write_url, config)

def persist_to_api_helper(url, config):
  headers = {'content-type': 'application/json'}
  config_persisted = False
  while config_persisted == False:
    r = requests.post(url, data=json.dumps(config), headers=headers)
    # retry persisting config if unsuccessful
    if r.status_code != 204:
      time.sleep(random.uniform(1, 2))
      continue
    else: 
      config_persisted = True
  return

# get_load_dist - expects JSON in the following format
  # {
  #   "max_weight": 20,
  #   "endpoints": [ # endpoints must be sorted by weight (ascending) - otherwise the results won't be deterministic
  #     {
  #       "path": "/dinosaurs",
  #       "name": "/dinosaurs",
  #       "weight": 5
  #     },
  #     ...
  #   ]
  # }
# if there is a need to use HTTP methods other than GET - this JSON structure must be changed to accommodate this
def get_load_dist(self):
  f = open('/mnt/locust/config/load.json',)
  
  data = json.load(f)
    
  if data["max_weight"] is None or data["endpoints"] is None or len(data["endpoints"]) == 0:
    sys.exit(f'Unable to read load distribution file')
    
  for endpoint in data["endpoints"]:
    if endpoint["path"] is None or endpoint["name"] is None or endpoint["weight"] is None:
      sys.exit(f'Invalid format of endpoints provided')

  f.close()

  return data