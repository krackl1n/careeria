#!/bin/sh
# Shared local renderer for MkDocs and standalone PlantUML commands.
set -eu
mode=${1:-render}
shift
case "$mode" in
    render|validate|pipe) ;;
    *) printf '%s\n' 'Expected render, validate or pipe' >&2; exit 2 ;;
esac
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
architecture_dir="$project_root/docs/architecture/c4-plant-uml"
version=${PLANTUML_VERSION:-1.2026.8}
image=${PLANTUML_IMAGE:-plantuml/plantuml:$version}

if [ -z "${PLANTUML_JAR:-}" ] && command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
    if [ "$mode" = pipe ]; then
        exec docker run --rm -i -v "$project_root:/workspace:ro" -w /workspace/docs "$image" "$@"
    fi
    if [ "$mode" = validate ]; then
        exec docker run --rm -v "$project_root:/workspace:ro" -w /workspace "$image" -failfast2 -checkonly "$@"
    fi
    exec docker run --rm -v "$project_root:/workspace" -w /workspace "$image" \
        -failfast2 -tsvg -o /workspace/docs/architecture/c4-plant-uml/images/generated "$@"
fi

if ! command -v java >/dev/null 2>&1; then
    printf '%s\n' 'Install Java or start Docker to render the architecture.' >&2
    exit 1
fi
jar=${PLANTUML_JAR:-$architecture_dir/.cache/plantuml-$version.jar}
if [ ! -f "$jar" ]; then
    if [ -n "${PLANTUML_JAR:-}" ]; then
        printf 'PlantUML JAR not found: %s\n' "$jar" >&2
        exit 1
    fi
    mkdir -p "$(dirname "$jar")"
    printf 'Downloading PlantUML %s...\n' "$version" >&2
    trap 'rm -f "$jar.download"' EXIT HUP INT TERM
    curl --fail --location --retry 2 --silent --show-error \
        "https://github.com/plantuml/plantuml/releases/download/v$version/plantuml-$version.jar" \
        -o "$jar.download"
    mv "$jar.download" "$jar"
    trap - EXIT HUP INT TERM
fi
if [ "$mode" = pipe ]; then
    cd "$project_root/docs"
    exec java -Djava.awt.headless=true -jar "$jar" "$@"
fi
if [ "$mode" = validate ]; then
    exec java -Djava.awt.headless=true -jar "$jar" -failfast2 -checkonly "$@"
fi
mkdir -p "$architecture_dir/images/generated"
# Smetana is bundled with PlantUML and does not require Graphviz.
exec java -Djava.awt.headless=true -jar "$jar" -failfast2 -tsvg \
    -o "$architecture_dir/images/generated" "$@"
