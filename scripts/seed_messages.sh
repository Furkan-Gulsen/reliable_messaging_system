#!/bin/bash

MONGODB_HOST="localhost"
MONGODB_PORT="27018" 
DB_NAME="message_system"
COLLECTION_NAME="messages"

echo "Seeding MongoDB with test data..."

docker-compose exec -T mongodb mongosh --eval "db.$COLLECTION_NAME.drop()" "$DB_NAME"

for i in {1..20}
do
  current_time=$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")
  random_number=$(( RANDOM % 900000000 + 100000000 ))  #
  random_phone="+90$random_number" 

  docker-compose exec -T mongodb mongosh "$DB_NAME" --eval "
    db.$COLLECTION_NAME.insertOne({
      content: 'Test message content #$i',
      status: 'unsent',
      to: '$random_phone',
      retry_count: 0,
      created_at: new Date('$current_time'),
      updated_at: new Date('$current_time')
    })
  "
done

echo "Successfully seeded $COLLECTION_NAME collection with 20 test messages"
