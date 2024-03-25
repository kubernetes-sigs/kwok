---
title: "Go Template in `kwok`"
---

# Notes on Go Template in `kwok`


The page provides a concise note on writing go templates in kwok CRs.


Currently, only `Stage` CR has go template based fields (`Spec.Next.StatusTemplate`).


You must follow [the go text template syntax] when writing the templates.
For predefined functions of go text template, please refer to [go text template functions].
Besides the built-in functions, `kwok` also supports [sprig template functions].

It is worth noting that the "context" (which is denoted by the period character `.` ) to a template in `kwok` is set to the 
referenced Kubernetes resource. 
For example, you can use `.metadata.name` in a template to obtain the corresponding Kubernetes resource name.



[the go text template syntax]: https://pkg.go.dev/text/template
[go text template functions]:  https://pkg.go.dev/text/template#hdr-Functions
[sprig template functions]: https://masterminds.github.io/sprig/
