# Demo vulnerability database (synthetic)

`Maven-all.zip` and `NuGet-all.zip` follow the [OSV schema](https://ossf.github.io/osv-schema/), but its records are
**invented** (prefix `TRAVERSEDEMO-`) to show how `--vulndb` works with `examples/demo-portfolio`.
It is **not** real vulnerability data. Never use it to assess real systems.

For real use, download the official dumps of the ecosystems you need from OSV.dev, for example:

    https://osv-vulnerabilities.storage.googleapis.com/Maven/all.zip
    https://osv-vulnerabilities.storage.googleapis.com/npm/all.zip
    https://osv-vulnerabilities.storage.googleapis.com/NuGet/all.zip
    https://osv-vulnerabilities.storage.googleapis.com/PyPI/all.zip
    https://osv-vulnerabilities.storage.googleapis.com/Packagist/all.zip
    https://osv-vulnerabilities.storage.googleapis.com/Go/all.zip

or use `scripts/fetch-osv.sh` / `scripts/fetch-osv.ps1` on a machine with Internet access.

The `commons-collections.jar` file in the demo portfolio is also a stub: it only contains Maven
metadata (`pom.properties`), so you can see how committed JARs are identified.
