import jwt, time, random

import sys, requests

# call 'api' microservice to get short living ocm token
def get_token(self):
  token_obtained = False
  ocm_token_attempt = 1

  while token_obtained == False and ocm_token_attempt < 4:
    r = requests.get('http://api:8099/ocm_token')

    # don't go any further it the response code wasn't StatusOK
    if r.status_code != 200:
      continue

    short_living_token = r.text.rstrip('\n')

    current_time = time.time()

    # confirming that token was retrieved
    if len(short_living_token) == 0:
      ocm_token_attempt = ocm_token_attempt + 1
      time.sleep(ocm_token_attempt * random.randint(5, 10))
    else:
      token_decoded = jwt.decode(short_living_token, options={'verify_signature': False})

      token_expiry = token_decoded['exp']

      if token_expiry > current_time:
        token_obtained = True
        
        self.client.headers = {'Content-Type': 'application/json', 'Authorization': 'Bearer ' + short_living_token}
  
  if token_obtained != True:
    sys.exit(f'Unable to obtain short-lived OCM token')
