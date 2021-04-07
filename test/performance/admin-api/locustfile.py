from locust import HttpUser, task
import requests, json, time, urllib3, os, sys

# disable ssl check on workers
urllib3.disable_warnings()

sso_url = os.getenv('ADMIN_API_SSO_AUTH_URL')
svc_acc_client_id = os.getenv('ADMIN_API_SVC_ACC_ID')
svc_acc_client_secret = os.getenv('ADMIN_API_SVC_ACC_SECRET')
if str(sso_url) == 'None' or str(svc_acc_client_id) == 'None' or str(svc_acc_client_secret) == 'None':
  sys.exit('Some of required params not specified (ADMIN_API_SSO_AUTH_URL or ADMIN_API_SVC_ACC_ID or ADMIN_API_SVC_ACC_SECRET)')

topic_name = 'topic01'

class MyUser(HttpUser):

  @task(1)
  def about(self):
    r = requests.post(sso_url, data={
                          'grant_type': 'client_credentials', 'client_id': svc_acc_client_id, 'client_secret': svc_acc_client_secret})
    if 'access_token' in r.text:
      json_response = json.loads(r.text)
      self.client.headers = {'Content-Type': 'application/json', 'Authorization': 'Bearer ' + json_response['access_token']}
      with self.client.get('/rest/topics', verify=False) as response:
        time.sleep(0.65)
        json_data = {'name': topic_name, 'settings': { 'numPartitions': 3, 'config': [{'key': 'cleanup.policy','value': 'delete'}]}}
        with self.client.post('/rest/topics', json=json_data, verify=False, catch_response=True ) as response:
          if response.status_code == 201 or response.status_code == 409:
            time.sleep(0.65)
            self.client.get(f'/rest/topics/{topic_name}', verify=False)
            time.sleep(0.65)
            delete_status = 0
            while delete_status != 200:
              with self.client.delete(f'/rest/topics/{topic_name}', verify=False, catch_response=True) as response:
                time.sleep(0.65)
                delete_status = response.status_code

