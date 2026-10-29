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
