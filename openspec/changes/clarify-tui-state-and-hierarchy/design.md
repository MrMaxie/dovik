# Approach

Retain Bubble Tea v2, Lip Gloss v2, the shared operator client, and the existing keyboard transitions. Make registry loading outcome explicit in the model and render it before selection-dependent content. Never derive successful emptiness solely from an empty item slice.

A successfully empty registry gets one content block: "No processes registered", a short explanation that registration uses the CLI, and refresh, setup help, and quit shortcuts. Setup help must show commands verified against the actual CLI, distinguish placeholders from literal arguments, and explain returning to the TUI and refreshing. Do not invent a registration wizard.

Loading uses a concise activity message. Failure uses a primary unavailable-state message with retry and quit; raw transport detail stays behind explicit diagnostic disclosure even when no process exists. A failed refresh after populated data must retain selection and output while clearly identifying unavailable or stale state.

For populated data, retain left navigation and a process identity/state/action summary above output at wide sizes. Render the summary without its own enclosing frame. Prefer alignment and a minimal separator between navigation and work area over a grid of full-height filled boxes. Use ordinary heading case and reserve the main accent for selection and contextual keyboard cues. Preserve textual lifecycle and stream labels, contrast, and no-color meaning.

Keep existing key meanings, including `l` for refresh. Hide selection and process-detail hints when they have no applicable subject; keep error-detail access discoverable on failure. Pending lifecycle operations name the action in progress. Preserve supported minimum dimensions and selection/output state across resize.

# Trade-offs

Improving the existing renderer is the smallest sufficient change. Focused Bubbles components could help future help or viewport maintenance, but are not needed for this outcome and are not part of this proposal. A tview migration would replace the event/presentation integration without addressing state hierarchy by itself.

Use separate empty and unavailable layouts instead of reserving the populated panel grid for every state. This deliberately changes the universal wide/compact layout requirement while preserving it for populated data.

Visual acceptance must include populated fixtures, not only the currently empty registry. Keep fixture registration and cleanup isolated from user-owned definitions.
