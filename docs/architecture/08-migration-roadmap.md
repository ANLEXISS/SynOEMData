# OEM migration roadmap

This roadmap was updated after M1.2. Each stage must preserve serialized
behavior and be validated against the baseline recorded in
[`16-modularization-validation.md`](16-modularization-validation.md).

1. **M1.3 — Core and Foundation split.** Separate cognitive snapshot/evidence
   algorithms from the Core-owned Universal Store behind tested consumer-owned
   ports. Do not change snapshot serialization or store ownership.
2. **M1.4 — Host/platform ports.** Specify interfaces for supervisor,
   persistent storage, network configuration, process health, media service and
   accelerator runtime. Keep systemd/Linux/RKNN as reference adapters; no
   behavior change until fakes reproduce current tests.
3. **M1.5 — Privilege and process composition.** Resolve Discovery vs standalone
   API deployment, document trust boundaries, and narrow root-only host
   operations behind a reviewed platform service. Preserve six current
   process boundaries while doing so.
4. **M1.6 — OEM device/media contract.** Define vendor-neutral capability,
   observation, availability, feedback, permissions and failure semantics.
   Implement no vendor adapter until one OEM hardware target and qualification
   plan are approved.
5. **M1.7 — Storage/runtime packaging.** Specify durable-store guarantees,
   schema migration/rollback, secrets provisioning and external model/runtime
   delivery; qualify a second OS/supervisor only after the ports are stable.
6. **M1.8 — Domain extraction.** Move one tested package boundary at a time,
   preserving `module synora` until import and contract stability support any
   later module split. Avoid copying algorithms to multiple owners. Voice and
   Automation remain unimplemented until real source behavior exists.
7. **M1.9 — OEM application layer.** Define branding/configuration and identity
   extension points without exposing internal state or bypassing server-side
   authorization.

Exit criteria for each stage: source traceability; no contract drift; green
baseline tests; explicit maturity and hardware qualification state; a rollback
path; and no undocumented host assumptions. The next recommended milestone is
M1.3, the Core/Foundation split described above.
