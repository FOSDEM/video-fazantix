import './style.css'

import favicon_dataurl from '../assets/favicon.png?url&inline';

import { UI } from './ui.ts'

function setup_favicon() {
    const link = document.querySelector<HTMLLinkElement>('#favicon')!;
    link.rel = 'icon';
    link.type = 'image/image/png';
    link.href = favicon_dataurl;

    const header_img = document.querySelector<HTMLImageElement>('#fazant')!;
    header_img.src = favicon_dataurl;
}

setup_favicon();

var ui = new UI();
ui.start()
