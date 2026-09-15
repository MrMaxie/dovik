# Third-party software

Dovik binaries statically link open-source Go modules. The authoritative dependency versions are recorded in `go.mod` and `go.sum`. Each module retains its own copyright and license; Dovik's Apache-2.0 license does not replace those terms.

Runtime dependencies include:

| Module family | License in the pinned source distribution |
| --- | --- |
| Charm Bubble Tea, Bubbles, Huh, Lip Gloss, and `charmbracelet/x` modules | MIT |
| Microsoft `go-winio` | MIT |
| Model Context Protocol Go SDK | Apache-2.0 and MIT transition terms as described by the module's `LICENSE` |
| Google JSON Schema for Go | Apache-2.0 |
| `golang.org/x/oauth2`, `x/sync`, `x/sys`, and `x/time` | BSD-3-Clause |
| Other transitive Go modules | Their respective license files in the pinned module source distributions |

The release pipeline checks that every module linked into `dovik`, `dovikd`, or the optional `gh` proxy exposes a license file before assembling an archive. Source and license locations can be resolved from the module paths and exact versions in `go.mod` and `go.sum`.
