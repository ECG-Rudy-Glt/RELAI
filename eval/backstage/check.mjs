import fs from 'node:fs';
import YAML from 'yaml';
import { entityEnvelopeSchemaValidator, makeValidator, componentEntityV1alpha1Validator, apiEntityV1alpha1Validator } from '@backstage/catalog-model';
const env = entityEnvelopeSchemaValidator();
const kinds = { Component: componentEntityV1alpha1Validator, API: apiEntityV1alpha1Validator };
const v = makeValidator();
let bad = 0, n = 0;
for (const f of process.argv.slice(2)) {
  for (const e of YAML.parseAllDocuments(fs.readFileSync(f, 'utf8')).map(d => d.toJSON())) {
    n++;
    try { env(e); } catch (x) { bad++; console.log(f, e.kind, 'ENVELOPE', x.message); continue; }
    for (const [fld, fn] of [['name', v.isValidEntityName], ['namespace', v.isValidNamespace]]) {
      const val = fld === 'name' ? e.metadata.name : (e.metadata.namespace ?? 'default');
      if (!fn(val)) { bad++; console.log(f, e.kind, 'bad', fld, val); }
    }
    try { const ok = await kinds[e.kind].check(e); if (!ok) { bad++; console.log(f, e.kind, e.metadata.name, 'KIND check false'); } } catch (x) { bad++; console.log(f, e.kind, 'KIND', x.message); }
  }
}
console.log(`${n} entities checked, ${bad} problems`);
