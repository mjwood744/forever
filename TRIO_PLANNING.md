# Trio planning sims (WoW Forever 3-box)

Local additions on top of wowsims/forever, used to plan the 3-box trios.

## What's added
- `sim/shaman/heals.go`: Shaman healing (Healing Wave, Lesser Healing Wave, Chain Heal jumping to the most injured, Riptide with +25% to the next Chain Heal, Ancestral Healing armour). Registered by the Restoration spec.
- `sim/paladin/mysim/`: Trio #1 (Rellyk Prot Paladin, Nyxx Disc Priest, Ashyn Arcane/Frost Mage) builds, stat weights, wanding, level-30 and red-team tests.
- `sim/warrior/mysim/`: Trio #2 (Skarr Prot Warrior) builds, with and without modelled healing.
- `sim/hunter/pet_tank.go`: a tanking pet's Beast Training (Great Stamina, Natural Armor, Pet Hardiness ranks in the hunter options), Endurance Training (+3% pet health a rank, assumed), Mend Pet (5 s channel, Improved Mend Pet cost cut) and Intimidation's 1,000 threat. Rogue Feint now uses rank 5.
- `sim/hunter/mysim/trio3_test.go`: Trio #3 (Hawke Hunter with the pet tanking, Wyldd Resto Druid, Shryke Rogue): TestT3Pet (families, training), TestT3Keep (Mend Pet thresholds), TestT3Rogue (rogue builds, Feint), TestT3Wyldd (druid builds).

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
go test -tags=with_db ./sim/hunter/mysim/ -run TestT3Pet -count=1 -v | grep TP
```
Each test prints one line per build and scenario (prefix shown in the grep).

## Updating from upstream
```
git fetch upstream && git rebase upstream/master
```
(`git remote add upstream https://github.com/wowsims/forever` the first time.)
