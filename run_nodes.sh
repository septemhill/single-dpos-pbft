#!/bin/bash

ID_BASE=0
NODES=21
PORT_BASE=10000

helpMsg() {
  echo ""
  echo $0 "[Id] [Nodes]" 
  echo "Id: Current machine ID base"
  echo "Nodes: How many nodes would created by the machine"
}

if [ "$1" == "" ];
then
  helpMsg
  exit 0
else
  ID_BASE=$1
fi

if [ "$2" != "" ];
then
  NODES=$2
fi

echo $ID_BASE $NODES

for i in $(seq 1 $NODES);
do
  ./single-dpos-pbft -p $(($PORT_BASE + $i - 1)) -i $(($ID_BASE * $NODES + $i - 1)) &
done
