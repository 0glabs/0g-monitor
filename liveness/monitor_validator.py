from web3 import Web3
from web3.middleware import geth_poa_middleware
import time
import datetime

import logging
from logging.handlers import TimedRotatingFileHandler
import pandas as pd


def filer(self):
    now = datetime.datetime.now()
    return log_file_name + now.strftime("%Y-%m-%d")


log_file_name = "log/validator_log_"
logger = logging.getLogger()
rotating_file_handler = TimedRotatingFileHandler(filename=log_file_name + datetime.datetime.now().strftime("%Y-%m-%d"), when="D", interval=1, backupCount=2)
rotating_file_handler.rotation_filename = filer
formatter = logging.Formatter("%(asctime)s %(name)s:%(levelname)s - %(message)s", "%Y-%m-%d %H:%M:%S")
rotating_file_handler.setFormatter(formatter)
logger.addHandler(rotating_file_handler)
logger.setLevel(logging.INFO)


rpcs = []
data = pd.read_csv("user-data/validator_rpcs.csv")
data.fillna('', inplace=True)


while True:
    for _, row in data.iterrows():
        rpc = row["validator_rpc"]
        validator = row['validator_address']
        
        if len(rpc) == 0:
            continue
        
        is_connected = False
        for r in rpc.split(','):
            r = r.strip()
            try:
                # Create a Web3 instance connected to the specified RPC URL
                w3_http = Web3(Web3.HTTPProvider(f'http://{r}', request_kwargs={'timeout': 3}))
                w3_https = Web3(Web3.HTTPProvider(f'https://{r}', request_kwargs={'timeout': 3}))

                # Check for connection to the Ethereum network
                if w3_http.is_connected() or w3_https.is_connected():
                    is_connected = True
                    break
            except Exception as e:
                continue
        
        if is_connected:
            logger.info(f"{rpc} of {validator} succeeded")
        else:
            logger.info(f"{rpc} of {validator} failed")

    time.sleep(30)
