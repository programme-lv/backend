"""
This script dumps the production database to a local file.
Requires pg_dump and aws cli to be installed.
The purpose is to replicate production db to local env for dev.
Reads the following env variables from .env:
POSTGRES_HOST,POSTGRES_PORT, POSTGRES_USER, POSTGRES_DB
POSTGRES_PASSWORD_SECRET_NAME
"""
import os
import subprocess
import json
import sys

def read_env_file(env_path='.env'):
    """
    Reads a .env file and returns a dictionary of environment variables.
    
    :param env_path: Path to the .env file.
    :return: Dictionary containing environment variables.
    """
    env_vars = {}
    try:
        with open(env_path, 'r') as file:
            for line in file:
                # Ignore comments and empty lines
                line = line.strip()
                if not line or line.startswith('#'):
                    continue
                if '=' not in line:
                    print(f"Warning: Skipping invalid line in .env file: {line}")
                    continue
                key, value = line.split('=', 1)
                # Remove surrounding quotes if present
                value = value.strip().strip('\'"')
                env_vars[key.strip()] = value
    except FileNotFoundError:
        print(f"Error: The .env file at path '{env_path}' was not found.")
        sys.exit(1)
    except Exception as e:
        print(f"Error reading .env file: {e}")
        sys.exit(1)
    return env_vars

def get_secret(secret_name, region='eu-central-1'):
    """
    Retrieves a secret from AWS Secrets Manager using the AWS CLI.
    
    :param secret_name: Name or ARN of the secret.
    :param region: AWS region where the secret is stored.
    :return: Dictionary containing the secret's key-value pairs.
    """
    try:
        # Execute the AWS CLI command to get the secret value
        cmd = [
            'aws',
            'secretsmanager',
            'get-secret-value',
            '--secret-id', secret_name,
            '--region', region
        ]
        result = subprocess.run(
            cmd,
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        secret_json = result.stdout
        secret_dict = json.loads(secret_json)
        
        if 'SecretString' in secret_dict:
            secret = secret_dict['SecretString']
            secret_data = json.loads(secret)
            return secret_data
        else:
            print("Error: SecretBinary is not supported in this script.")
            sys.exit(1)
    except subprocess.CalledProcessError as e:
        print(f"Error retrieving secret from AWS Secrets Manager: {e.stderr}")
        sys.exit(1)
    except json.JSONDecodeError as e:
        print(f"Error parsing JSON from secret: {e}")
        sys.exit(1)
    except Exception as e:
        print(f"Unexpected error retrieving secret: {e}")
        sys.exit(1)

def execute_pg_dump(host, port, user, db_name, password, sslmode, dump_file='./mydb.dump'):
    """
    Executes the pg_dump command with the provided parameters.
    
    :param host: PostgreSQL host.
    :param port: PostgreSQL port.
    :param user: PostgreSQL username.
    :param db_name: Name of the database to dump.
    :param password: PostgreSQL password.
    :param sslmode: SSL mode for the connection.
    :param dump_file: Path to the output dump file.
    """
    # Construct the pg_dump command
    cmd = [
        'pg_dump',
        '-h', host,
        '-p', str(port),
        '-U', user,
        '-F', 'c',        # Custom format
        '-b',             # Include large objects
        '-v',             # Verbose mode
        '-f', dump_file,  # Output file
        db_name
    ]
    
    # Set the PGPASSWORD environment variable for authentication
    env = os.environ.copy()
    env['PGPASSWORD'] = password
    
    try:
        print("Starting pg_dump...")
        process = subprocess.Popen(
            cmd,
            env=env,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            bufsize=1,
            universal_newlines=True
        )
        
        # Stream output in real-time
        for line in process.stdout:
            print(line, end='')
            
        process.stdout.close()
        return_code = process.wait()
        
        if return_code != 0:
            print(f"pg_dump failed with return code {return_code}")
            sys.exit(1)
            
        print("pg_dump completed successfully.")
        
    except FileNotFoundError:
        print("Error: pg_dump command not found. Please ensure it is installed and in your PATH.")
        sys.exit(1)
    except Exception as e:
        print(f"Unexpected error during pg_dump: {e}")
        sys.exit(1)

def main():
    # Step 1: Read environment variables from .env file
    env_vars = read_env_file('../.env')
    
    required_vars = [
        'POSTGRES_HOST',
        'POSTGRES_PORT',
        'POSTGRES_USER',
        'POSTGRES_DB',
        'POSTGRES_PASSWORD_SECRET_NAME'
    ]
    
    missing_vars = [var for var in required_vars if var not in env_vars]
    if missing_vars:
        print(f"Error: Missing required environment variables: {', '.join(missing_vars)}")
        sys.exit(1)
    
    # Extract necessary variables
    host = env_vars['POSTGRES_HOST']
    port = env_vars['POSTGRES_PORT']
    user = env_vars['POSTGRES_USER']
    db_name = env_vars['POSTGRES_DB']
    sslmode = env_vars.get('POSTGRES_SSLMODE', 'require')
    secret_name = env_vars['POSTGRES_PASSWORD_SECRET_NAME']
    
    # Step 2: Retrieve secret using AWS CLI
    secret = get_secret(secret_name, region='eu-central-1')  # Adjust region if necessary
    
    # Extract password from the secret
    password = secret.get('password')
    secret_username = secret.get('username')
    
    if not password:
        print("Error: Password not found in the retrieved secret.")
        sys.exit(1)
    
    # Optional: Verify that the username from the secret matches the POSTGRES_USER
    if secret_username and secret_username != user:
        print("Warning: Username from secret does not match POSTGRES_USER.")
        print(f"Secret Username: {secret_username}")
        print(f"POSTGRES_USER: {user}")
    
    # Step 3: Execute pg_dump
    execute_pg_dump(
        host=host,
        port=port,
        user=user,
        db_name=db_name,
        password=password,
        sslmode=sslmode,
        dump_file='./prod-pg.dump'
    )

if __name__ == '__main__':
    main()
