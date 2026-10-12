import { createRoot } from 'react-dom/client';
import { Rehearsal } from './Rehearsal';
import '../src/app/style.css';
// This entry is built separately; it never imports the financial application.
createRoot(document.getElementById('root')!).render(<Rehearsal />);
