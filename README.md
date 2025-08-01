# Dyson

This is a program to perform calculations about production chains in Dyson Sphere Program.  Right now it can
list the dependencies of a given set of items, or calculate what can be built from a given set of items.

### Chain command

```
$ ./dyson chain "Mining Machine"
Mining Machine: Circuit Board, Gear, Iron Ingot, Magnetic Coil
Circuit Board: Copper Ingot, Iron Ingot
Gear: Iron Ingot
Iron Ingot: Iron Ore
Magnetic Coil: Copper Ingot, Magnet
Copper Ingot: Copper Ore
Iron Ore: <produced by mine>
Magnet: Iron Ore
Copper Ore: <produced by mine>`
```

In this case we have asked what resources need to exist in order to create a Mining Machine.  The program has replied
that we need to make circuit boards, gears, etc., and all their dependencies.

### Makes command

```
$ ./dyson makes "Iron Ore" "Coal"
Iron Ore: <unknown>
Coal: <unknown>
Iron Ingot: Iron Ore
Energetic Graphite: Coal
Proliferator Mk. I: Coal
Magnet: Iron Ore
Diamond: Energetic Graphite
Combustible Unit: Coal
Steel: Iron Ingot
Gear: Iron Ingot
Conveyor Belt Mk. I: Gear, Iron Ingot
Proliferator Mk. II: Diamond, Proliferator Mk. I
```

This command shows what we can make from a given set of inputs.  In this case we only have iron ore and coal, and want
to know what can be produced from them.

### Diff command

```
$ ./dyson diff --old "Iron Ore" --old "Coal" --new "Stone"
Stone: <unknown>
Stone Brick: Stone
Glass: Stone
Prism: Glass
Silicon Ore: Stone
Foundation: Steel, Stone Brick
Depot Mk. I: Iron Ingot, Stone Brick
Depot Mk. II: Steel, Stone Brick
Storage Tank: Glass, Iron Ingot, Stone Brick
High-Purity Silicon: Silicon Ore
Crystal Silicon: High-Purity Silicon
```

This command shows us what can be newly produced if we add a given resource to some already-existing resources.  In
this case, we already have iron ore and coal as in the previous example, but we now add stone.  The output is only
those items which we can now make with stone, but which we couldn't make before.  (This is intended for mall planning.)

One problem that arises here is that sometimes you can make things through an inefficient process.  For example, in
this case we can make silicon ore from stone, but we don't actually _want_ to make silicon ore from stone.  To handle
such cases, we can do:

```
$ ./dyson diff --old "Iron Ore" --old "Coal" --new "Stone" --exclude-new "Silicon Ore"
Stone: <unknown>
Stone Brick: Stone
Glass: Stone
Prism: Glass
Foundation: Steel, Stone Brick
Depot Mk. I: Iron Ingot, Stone Brick
Depot Mk. II: Steel, Stone Brick
Storage Tank: Glass, Iron Ingot, Stone Brick
```

This blocks the ability to create silicon ore as a new output, thus eliminating it and all its downstream products.

