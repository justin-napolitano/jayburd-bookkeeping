import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
// Source fences prevent a future foundation expansion from claiming a second SQL instance or secret values.
test('shared foundation has no SQL instance, operator, secret payload or live workload resource',()=>{
 const root='infra/gcp/shared-production-foundation/';
 const source=['main.tf','resources.tf'].map(f=>readFileSync(root+f,'utf8')).join('\n');
 assert.doesNotMatch(source,/resource\s+"google_sql_database_instance"|resource\s+"google_secret_manager_secret_version"|database_roles\s*=|resource\s+"google_cloud_run|resource\s+"google_cloud_scheduler/);
 assert.doesNotMatch(source,/roles\/owner|roles\/editor|roles\/storage\.objectAdmin/);
 assert.match(source,/name\s*=\s*"bookkeeping_prod"/);
 assert.match(source,/instance\s*=\s*"jayburd-dev"/);
});
// Imperative imports expand configuration before resource-derived collections are known.
test('import graph expands IAM from static keys before resource creation',()=>{
 const source=readFileSync('infra/gcp/shared-production-foundation/resources.tf','utf8');
 assert.doesNotMatch(source,/for_each\s*=\s*google_|for_each[^\n]*keys\(google_/);
 assert.match(source,/setproduct\(\["api", "sync"\], local\.provider_destinations\)/);
 assert.match(source,/for_each\s*=\s*local\.provider_keys/);
 assert.match(source,/for_each\s*=\s*local\.provider_destinations/);
});
