import { createCardCanvas } from "./renderer";

async function loadCards(){
    const soldier1 = await createCardCanvas({
        base:'./sprites/BaseFloyd.png',
        overlays: ['./sprites/green-cap.png'],
    });
}




document.body.append