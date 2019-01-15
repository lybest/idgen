#! /bin/sh
for((i=1;i<=100;i++)){
	nohup ./idgen -svrid $i &
}

