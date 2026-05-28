# Citations

Every paper and production system referenced in chrono.

## Foundational papers

**Lamport (1978).** "Time, Clocks, and the Ordering of Events in a
Distributed System." *Communications of the ACM* 21(7), 558–565.
https://lamport.azurewebsites.net/pubs/time-clocks.pdf

The paper that started it all. Defines the happens-before relation,
the logical clock algorithm, and the "no global time" framing every
subsequent paper builds on. Implemented in `lamport/`; reproduced in
`proofs/lamport_1978_test.go`.

**Fidge (1988).** "Timestamps in Message-Passing Systems That Preserve
the Partial Ordering." *Australian Computer Science Communications*
10(1), 56–66.

**Mattern (1989).** "Virtual Time and Global States of Distributed
Systems." *Workshop on Parallel and Distributed Algorithms*.
https://www.vs.inf.ethz.ch/publ/papers/VirtTimeGlobStates.pdf

Independent inventions of vector clocks. Mattern's exposition is more
widely cited. Implemented in `vector/`.

**Preguiça, Baquero, Almeida, Fonte, Gonçalves (2010).** "Dotted
Version Vectors: Logical Clocks for Optimistic Replication."
arXiv:1011.5808. https://arxiv.org/abs/1011.5808

Decouples the "actor" from the "node," solving the dynamic-membership
problem that plagued classic vector clocks in Dynamo-style systems.
Implemented in `dotted/`; reproduced in `proofs/riak_siblings_test.go`.

**Kulkarni, Demirbas, Madappa, Avva, Leone (2014).** "Logical Physical
Clocks and Consistent Snapshots in Globally Distributed Databases."
*OPODIS*. https://cse.buffalo.edu/tech-reports/2014-04.pdf

Hybrid Logical Clocks. The paper bridges Lamport's logical clocks with
physical time, giving timestamps that are causal *and* close to wall
time. Implemented in `hlc/`; reproduced in `proofs/hlc_etcd_test.go`.

**Corbett et al. (2012).** "Spanner: Google's Globally-Distributed
Database." *OSDI*.
https://research.google/pubs/spanner-googles-globally-distributed-database-2/

The TrueTime paper. Introduces bounded-uncertainty clocks and the
commit-wait protocol for external consistency. Implemented in
`truetime/`; reproduced in `proofs/spanner_external_test.go`.

## Production systems referenced

- **Riak:** uses dotted version vectors since 2.0. https://riak.com/

- **CockroachDB:** HLC for transaction timestamps and snapshot reads.
  Their "Living Without Atomic Clocks" blog post is essential
  background.
  https://www.cockroachlabs.com/blog/living-without-atomic-clocks/

- **YugabyteDB:** HLC, same model as CockroachDB.
  https://docs.yugabyte.com/

- **MongoDB:** HLC variant for cross-shard transactions since 4.0.

- **Amazon Dynamo (2007 paper):** vector clocks for sibling detection
  in a leaderless store. Successor systems (DynamoDB) moved to other
  mechanisms, but the paper is canonical reading.
  https://www.allthingsdistributed.com/files/amazon-dynamo-sosp2007.pdf

- **Google Spanner:** TrueTime in production. Spanner papers and the
  Cloud Spanner documentation describe the system as deployed.
  https://cloud.google.com/spanner

## Related reading

- **Schwarz & Mattern (1994).** "Detecting Causal Relationships in
  Distributed Computations: In Search of the Holy Grail." Survey of
  causal-ordering mechanisms; useful context for why these primitives
  exist.

- **Kleppmann (2017).** *Designing Data-Intensive Applications*, Ch. 8
  & 9. The most accessible single-source introduction to causality,
  consistency, and time in distributed systems.

## NTP protocol references

- **RFC 4330**: Simple Network Time Protocol (SNTP). The protocol
  `truetime/ntp_source.go` implements.
- **RFC 5905**: Network Time Protocol v4 (the full one; we implement
  the SNTP subset).
