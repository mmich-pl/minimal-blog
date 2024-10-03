#!/bin/bash
# If you have credentials for your DB uncomment the following two lines
#USER_NAME='user_name'
#PASSWORD='user_password'

sleep 10

# If you have credentials for your DB use: while ! cqlsh scylla -u "${USER_NAME}" -p "${PASSWORD}" -e 'describe cluster' ; do
while ! cqlsh scylla -e 'describe cluster' ; do
     sleep 5
done

for cql_file in ./scylla_scripts/*.cql;
do
# If you have credentials on your db use this line cqlsh scylla -u "${USER_NAME}" -p "${PASSWORD}" -f "${cql_file}" ;
  cqlsh scylla -f "${cql_file}" ;
done
