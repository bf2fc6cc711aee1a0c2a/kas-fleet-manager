import json, random, requests, string, subprocess, time

# golang API helper urls used to persist config
svc_acc_id_write_url = 'http://api:8099/write_svc_acc_id'
kafka_id_write_url = 'http://api:8099/write_kafka_id'
kafka_config_write_url = 'http://api:8099/write_kafka_config'

# check if container id of the container making the request was elected
# to create kafka requests
def check_kafka_create_enabled():
  get_container_id_output = subprocess.check_output("cat /proc/self/cgroup | head -n 1 | cut -d '/' -f3", shell=True)
  container_id = get_container_id_output.decode('utf-8')
  container_info = {
            'containerId': container_id,
          }
  headers = {'content-type': 'application/json'}
  container_id_sent = False
  while container_id_sent == False:
    r = requests.post('http://api:8099/kafka_create_container_id', data=json.dumps(container_info), headers=headers)
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

# generates description for service account
def svc_acc_description():
  return 'Created by the performance test suite of the API'

# determines if the id of the service account was created by performance test
def created_by_perf_test(svc_acc_id, items):
  created_by_performance_test = False
  for item in items:
    if 'id' in item:
      if item['id'] in svc_acc_id:
        if 'description' in item:
          if svc_acc_description() in item['description']:
            created_by_performance_test = True
            break
  return created_by_performance_test

# random_string is used to create kafka_request names
def random_string(length):
   return ''.join(random.choice(string.ascii_lowercase) for i in range(length))

# random_status - get random status of kafka_cluster
def random_status():
  return random.choice(['accepted', 'preparing', 'provisioning', 'ready'])

# kafka_json is used to create json payload 
def kafka_json():
  return {
           'cloud_provider': 'aws',
           'region': 'us-east-1',
           'name': random_string(15),
           'multi_az': True
         }

# get_random_search query to get with search param
def get_random_search():
  return f'cloud_provider = aws and status = {random_status()} and name <> {random_string(15)}'

# get random id from list
def get_random_id(list):
  return random.choice(list)

# generate random service account id
def generate_random_svc_acc_id():
  return f'{random_alfa_numeric(7)}-{random_alfa_numeric(4)}-{random_alfa_numeric(4)}-{random_alfa_numeric(4)}-{random_alfa_numeric(11)}'

# generate random service account client ID
def generate_random_svc_acc_client_id():
  return f'srvc-acct-{random_alfa_numeric(8)}-{random_alfa_numeric(4)}-{random_alfa_numeric(4)}-{random_alfa_numeric(4)}-{random_alfa_numeric(12)}'

# generate random service account secret
def generate_random_svc_acc_secret():
  return f'{random_alfa_numeric(8)}-{random_alfa_numeric(4)}-{random_alfa_numeric(4)}-{random_alfa_numeric(4)}-{random_alfa_numeric(12)}'

# get random alpha-numeric string
def random_alfa_numeric(length):
  return ''.join(random.choices(string.ascii_lowercase + string.digits, k=length))

def safe_delete(list, item):
  if item in list:
    list.remove(item)

# generate and return service account json
def svc_acc_json(base_url):
  svc_acc_id = generate_random_svc_acc_id()
  return {
           'id': svc_acc_id,
           'href': f'{base_url}/serviceaccounts/{svc_acc_id}',
           'clientID': generate_random_svc_acc_client_id(),
           'name': random_string(10),
           'description': svc_acc_description(),
           'clientSecret': generate_random_svc_acc_secret()
         }

# get details of service account associated with kafka
def get_svc_account_for_kafka(collection, kafka_id):
  for item in collection:
    if 'kafka_id' in item and item['kafka_id'] == kafka_id:
      return item
  return []

# send kafka bootstrap URL and svc acc credentials to the helper API
# which will persist it to a text file
def persist_kafka_config(bootstrapURL, svc_acc_json):
  config = {
             'bootstrapURL': bootstrapURL,
             'username': svc_acc_json['clientID'],
             'password': svc_acc_json['clientSecret'],
           }
  persist_to_api_helper(kafka_config_write_url, config)

def persist_kafka_id(kafka_id):
  config = {
             'kafkaId': kafka_id,
           }
  persist_to_api_helper(kafka_id_write_url, config)

def persist_service_account_id(svc_acc_id):
  config = {
             'serviceAccountId': svc_acc_id,
           }
  persist_to_api_helper(svc_acc_id_write_url, config)

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
