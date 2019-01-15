#! /bin/sh

while [ 1 ];do
	ps -ef | grep idgen | grep -v grep | cut -c 9-15 | head -n 1 | xargs kill -9	
	sleep 20
done
