"""MCP server Lexora: kasih konteks project ke session Claude baru."""
from pathlib import Path
from mcp.server.fastmcp import FastMCP

PLAN = Path(__file__).resolve().parent.parent / "PLAN"
mcp = FastMCP("lexora")


def _docs() -> dict[str, Path]:
    return {p.stem: p for p in PLAN.glob("*.md")}


@mcp.tool()
def lexora_context() -> str:
    """Ringkasan project Lexora + daftar dokumen PLAN. Panggil di awal session."""
    readme = PLAN / "00-README.md"
    daftar = "\n".join(f"- {name}" for name in sorted(_docs()))
    isi = readme.read_text(encoding="utf-8") if readme.exists() else "(README belum ada)"
    return f"{isi}\n\n## Dokumen PLAN tersedia (pakai read_plan)\n{daftar}"


@mcp.tool()
def read_plan(doc: str) -> str:
    """Baca satu dokumen PLAN. doc = nama file tanpa .md, mis. '01-PRD' atau '03-ERD'."""
    docs = _docs()
    if doc not in docs:
        return f"'{doc}' tidak ada. Pilihan: {', '.join(sorted(docs))}"
    return docs[doc].read_text(encoding="utf-8")


@mcp.prompt()
def mulai_fase(fase: str) -> str:
    """Skill siap-pakai: mulai kerjakan satu fase Lexora sesuai PLAN."""
    return (
        f"Panggil tool lexora_context dulu biar paham project. "
        f"Lalu kerjakan Fase {fase}: baca detailnya via read_plan('05-PLANNING') "
        f"dan ikuti aturan di read_plan('08-AGENT-BRIEF')."
    )


if __name__ == "__main__":
    mcp.run()
