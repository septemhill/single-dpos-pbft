 kill -9 $(ps -ef | grep single | awk '{print $2}')
