# I1 Model Candidate Comparison

**Status:** EXPERIMENTAL comparison; no production model selection  
**Data/measurements:** The only measured candidate is the 31-parameter synthetic I1 proof of concept. All other resource values below are estimates, not benchmarks.

## Candidates

| Candidate | Expressiveness and interpretation | Data/training cost | Inference, memory, quantization, embedded fit | Failure modes and I1 disposition |
|---|---|---|---|---|
| **A — structured features + multitask MLP** | Learns nonlinear combinations of typed state features. A single vector is weak for long event histories and variable-size graphs. Heads can be separate or share hidden layers; sharing must be evaluated, not assumed. Inspectable feature names/masks and task-head outputs help, but weights are not rule explanations. | Requires labeled next-state/transition/anomaly examples with group-safe splits. Small MLP training is straightforward. Multitask loss interference and label quality remain unmeasured. | Small dense layers; compute roughly O(input×hidden + hidden×heads). Parameter storage is small relative to large pretrained models; FP16/INT8 are plausible for dense operators but need actual toolchain tests. Best embedded candidate of the three in principle; no RK3588 measurement. | Can collapse unknowns, overfit a generator, be miscalibrated, and conflate tasks. Candidate for I4 only after I2/I3. The I1 toy MLP validates one narrow learnable relation, not a shared production backbone. |
| **B — structured + explicit temporal/topological encoders + MLP** | Adds event order, bounded time context, and graph relations. Better suited to sequence/adjacency tasks than flattening an arbitrary history/graph; does not guarantee better reasoning. Interpretability depends on preserving source-aligned masks and relation features. | Needs sequence labels and multiple home/topology groups; leakage control is more demanding. Temporal and graph components must be ablated separately. Training complexity moderate. | Example future candidate: small GRU/temporal convolution plus one or two graph aggregation steps, then compact heads. Cost scales with history window T, hidden width d, and edges E; approximately O(Td² + Ed) for a small recurrent/aggregation pass. Memory depends on bounded history and activations. Dense/recurrent quantization may be feasible; operator support and latency on ARM64/RK3588 are unverified. | More moving parts, graph-ID leakage, ordering mistakes, unbounded windows, and increased calibration/latency risk. Preferred research direction for tasks B, but only if grouped evaluation beats Candidate A and transition-table baselines. |
| **C — optional pretrained language backbone with specialized heads/adapters, e.g. Qwen3-0.6B** | Can represent language and text-rich context; an autoregressive text model is a poor default for routine structured state and may hide values in prompts. Adapters/heads do not make unsupported factual grounding safe. | Highest data, training, prompt/adapter, evaluation, and artifact management burden. Requires provenance-safe text conversion and leakage/identity review. No weight was downloaded and no experiment was run. | For a nominal 0.6B-parameter candidate, raw weight storage is approximately 2.4 GB at FP32, 1.2 GB at FP16, or 0.3 GB at 4-bit before quantization metadata, tokenizer, runtime, KV cache, and workspace; these are arithmetic estimates. Embedded CPU/NPU fit and quantized operator support are unverified and likely substantially harder than A/B. | Hallucinated facts, prompt-induced semantic drift, privacy exposure, large memory/latency, and version/toolchain dependence. Explicitly not the default. Consider only if measured language-specific task needs remain after structured candidates. |

## Existing implementation is not a candidate result

The V1 CPU bundle is an inference-only format with the frozen 86-value V1 encoder and separate per-head artifacts. The V3 CPU candidate has a shared backbone and fixed heads over the separate V3 86-value encoder, with validation requiring dry-run and prohibiting promotion/physical effects. Source search found no complete cognitive training/dataset pipeline producing these artifacts. Their interfaces are useful compatibility constraints, not quality evidence or proof that A/B has been qualified.

## Selection hypothesis

1. Use simple baselines first.
2. Build the I2 representation with explicit validity masks before choosing a learned backbone.
3. Start I4 comparisons with a compact shared MLP (A) and a small temporal/topological variant (B), plus separate task models as controls.
4. Evaluate whether task losses interfere with one another.
5. Keep C only if a task requires language reasoning that structured A/B cannot express and resource/privacy prerequisites are approved.

No model is selected for production. Training success, synthetic held-out success, task validation, and embedded qualification remain separate gates.
