# 1-node setup
#halmidi-node0: ./build/bin/halmidi start --dev-config ./assets/dev_cluster/halmidi.node0.conf --nodename node0 --id 1 --peers 1=http://127.0.0.1:9502 --port 9501

# 2-node setup
# halmidi-node0: ./build/bin/halmidi start --dev-config ./assets/dev_cluster/halmidi.node0.conf --nodename node0 --id 1 --peers 1=http://127.0.0.1:9502,2=http://127.0.0.1:9512 --port 9501
# halmidi-node1: ./build/bin/halmidi start --dev-config ./assets/dev_cluster/halmidi.node1.conf --nodename node1 --id 2 --peers 1=http://127.0.0.1:9502,2=http://127.0.0.1:9512 --port 9511

# 3-node setup
halmidi-node0: ./build/bin/halmidi start --dev-config ./assets/dev_cluster/halmidi.node0.conf --nodename node0 --id 1 --peers 1=http://127.0.0.1:9502,2=http://127.0.0.1:9512,3=http://127.0.0.1:9522 --port 9501
halmidi-node1: ./build/bin/halmidi start --dev-config ./assets/dev_cluster/halmidi.node1.conf --nodename node1 --id 2 --peers 1=http://127.0.0.1:9502,2=http://127.0.0.1:9512,3=http://127.0.0.1:9522 --port 9511
halmidi-node2: ./build/bin/halmidi start --dev-config ./assets/dev_cluster/halmidi.node2.conf --nodename node2 --id 3 --peers 1=http://127.0.0.1:9502,2=http://127.0.0.1:9512,3=http://127.0.0.1:9522 --port 9521
