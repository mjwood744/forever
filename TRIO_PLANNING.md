# Trio planning sims (WoW Forever 3-box)

Local additions on top of wowsims/forever, used to plan the 3-box trios.

## What's added
- `sim/shaman/heals.go`: Shaman healing (Healing Wave, Lesser Healing Wave, Chain Heal jumping to the most injured, Riptide with +25% to the next Chain Heal, Ancestral Healing armour). Registered by the Restoration spec.
- `sim/paladin/mysim/`: Trio #1 (Rellyk Prot Paladin, Nyxx Disc Priest, Ashyn Arcane/Frost Mage) builds, stat weights, wanding, level-30 and red-team tests.
- `sim/warrior/mysim/`: Trio #2 (Skarr Prot Warrior) builds, with and without modelled healing.

## Setup (once)
```
pip install grpcio-tools
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
python3 -m grpc_tools.protoc -I=./proto -I=$(python3 -c "import grpc_tools,os;print(os.path.dirname(grpc_tools.__file__)+'/_proto')") \
  --go_opt=Mgoogle/protobuf/descriptor.proto=google.golang.org/protobuf/types/descriptorpb --go_out=./sim/core ./proto/*.proto
```

## Run a test
```
go test -tags=with_db ./sim/paladin/mysim/ -run TestRed -count=1 -v | grep RED
go test -tags=with_db ./sim/warrior/mysim/ -run TestSkarrHealed -count=1 -v | grep SH
```
Each test prints one line per build and scenario (prefix shown in the grep).

## Updating from upstream
```
git fetch upstream && git rebase upstream/master
```
(`git remote add upstream https://github.com/wowsims/forever` the first time.)
