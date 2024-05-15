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


while True:
    for _, row in data.iterrows():
        rpc = row["rpc"]
        validator = row['validator']
        try:
            # Create a Web3 instance connected to the specified RPC URL
            w3 = Web3(Web3.HTTPProvider(rpc))

            # Inject PoA middleware for networks using Proof of Authority consensus
            w3.middleware_onion.inject(geth_poa_middleware, layer=0)

            # Check for connection to the Ethereum network
            if not w3.is_connected():
                logger.error(f"{rpc} of {validator} failed")
            else:
                logger.info(f"{rpc} of {validator} succeeded")
        except Exception as e:
            logger.info(f"{rpc} of {validator} failed")
        

    time.sleep(60)
