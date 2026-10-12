import { publicConfig } from '../../../server/public-config';
export const runtime = 'nodejs';
export const dynamic = 'force-dynamic';
/** Evaluate public identifiers at request time so one image can be configured without rebuilding. */
export function GET() { return publicConfig(process.env); }
