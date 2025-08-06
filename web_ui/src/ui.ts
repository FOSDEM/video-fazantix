interface ConfigResponse {
    stages: Array<Stage>
    scenes: Array<Scene>
}

interface Stage {
    Name: string
    PreviewFor: string
    Type: StageType
    Preview: Stage | undefined
}

interface Scene {
    Code: string
    Tag: string
    Label: string
}

type StageType = "preview" | "aux" | "program"

function switchScene(stage: string, scene: string) {
	console.log("Switching to", scene, "on stage", stage)
	fetch("/api/scene/"+stage+"/"+scene).then()
}

function makeAuxControls(_aux: Stage) {
    const controlbox = document.createElement("section")
    controlbox.classList.add("control")

    const transitionCut = document.createElement("button")
    transitionCut.classList.add("btn")
    transitionCut.classList.add("yellow")
    transitionCut.innerText = "CUT"
    controlbox.appendChild(transitionCut)

    const transitionAuto = document.createElement("button")
    transitionAuto.classList.add("btn")
    transitionAuto.classList.add("yellow")
    transitionAuto.classList.add("active")
    transitionAuto.innerText = "AUTO"
    controlbox.appendChild(transitionAuto)
    return controlbox
}
function makeMEControls(program: Stage, preview: Stage) {
    const controlbox = document.createElement("section")
    controlbox.classList.add("control")

    const transitionCut = document.createElement("button")
    transitionCut.classList.add("btn")
    transitionCut.innerText = "CUT"
    controlbox.appendChild(transitionCut)

    const transitionAuto = document.createElement("button")
    transitionAuto.classList.add("btn")
    transitionAuto.innerText = "AUTO"
    transitionAuto.addEventListener("click", function (event) {
        event.preventDefault()
        const currentProgram = document.querySelector<HTMLButtonElement>("button[data-stage="+program.Name+"].active")!
        const currentPreview = document.querySelector<HTMLButtonElement>("button[data-stage="+preview.Name+"].active")!
        switchScene(program.Name, currentPreview.dataset.scene!)
        switchScene(preview.Name, currentProgram.dataset.scene!)
    })
    controlbox.appendChild(transitionAuto)
    return controlbox
}

function makeButtons(scenes: Array<Scene>, stage: Stage) {
    const buttonBox = document.createElement("section")
    buttonBox.classList.add("buttonbox")

    scenes.forEach(scene => {
        const button = document.createElement("button")
        button.innerText = scene.Tag
        button.title = scene.Label
        button.dataset.stage = stage.Name
        button.dataset.scene = scene.Code
        buttonBox.appendChild(button)

        button.addEventListener("click", function() {
            switchScene(stage.Name, scene.Code)
        })
    })
    return buttonBox
}
function makeStages() {
	const stages = document.getElementById("stages")!
	fetch('/api/config').then(response => response.json()).then((response: ConfigResponse) => {
        let stageList: Array<Stage> = []
        // First get all the program/aux sources
        response.stages.forEach(stage => {
            if(stage.PreviewFor === "") {
                stage.Type = "aux"
                stageList.push(stage)
            }
        })
        // Insert all the preview buses in the program bus it's linked to
        response.stages.forEach(stage => {
            if(stage.PreviewFor !== "") {
                stage.Type = "preview"
                stageList.forEach(programStage => {
                    if(programStage.Name !== stage.PreviewFor) {
                        return
                    }
                    programStage.Type = "program"
                    programStage.Preview = stage
                })
            }
        })
        stageList.forEach(stage => {
			const stageBox = document.createElement("section")
			stageBox.dataset.stage = stage.Name
            stageBox.dataset.previewFor = stage.PreviewFor
            stageBox.dataset.type = stage.Type
            if (stage.PreviewFor !== "") {
                stageBox.classList.add("preview")
            } else {
                stageBox.classList.add("program")
            }
			stages.appendChild(stageBox)

			const stageHeader = document.createElement("header")
			stageHeader.innerText = stage.Name
			stageBox.appendChild(stageHeader)

            const buttonbox = document.createElement("section")
            buttonbox.classList.add("buttons")
            stageBox.appendChild(buttonbox)

            const buttons = makeButtons(response.scenes, stage)
            buttons.classList.add("program")
            buttonbox.appendChild(buttons)
            if(stage.Type === "program") {
                console.log("Make program for", stage)
                const preview = makeButtons(response.scenes, stage.Preview!)
                preview.classList.add("preview")
                buttonbox.appendChild(preview)
                stageBox.appendChild(makeMEControls(stage, stage.Preview!))
            } else if (stage.Type === "aux") {
                stageBox.appendChild(makeAuxControls(stage))
            }
		})
        makeWebsocket()
        })
    	.catch(err => console.error(err))
}
function makeWebsocket() {
        socket = new WebSocket("/api/ws")
        socket.onopen = _event => {
            document.getElementById("logo")!.style.color = "white"
            console.log("Connected to websocket")
        }

        socket.onmessage = function(event) {
		const message = JSON.parse(event.data)
		if ("fps" in message) {
			const fpsElem = document.getElementById("stat-fps")!
			fpsElem.innerText = `${Math.round(message["fps"] * 10) / 10}`
		}
		if ("ws_clients" in message) {
			const clientsElem = document.getElementById("stat-clients")!
			clientsElem.innerText = message["ws_clients"]
		}
        if ("Event" in message) {
            switch(message["Event"]) {
                case "set-scene":
                    document.querySelectorAll<HTMLButtonElement>("button[data-stage="+message["Stage"]+"]").forEach((el) =>  {
                        el.classList.remove("active")
                    })
                    let btn = document.querySelector<HTMLButtonElement>("button[data-stage="+message["Stage"]+"][data-scene="+message["Scene"]+"]")!
                    if(btn === undefined) {
                        console.error("Could not find button for stage '"+message["Stage"]+"' scene '"+message["Scene"]+"'")
                    }
                    btn.classList.add("active")
                    break
            }
        }
        }

        socket.onclose = _event => {
		console.error("Websocket disconnected")
		document.getElementById("logo")!.style.color = "red"
		setTimeout(()=>{
			makeWebsocket()
		}, 2000)
        }
}

let socket

export class UI {
    start() {
        // FIXME: reorganise code to not have global state and non-encapsulated async events
        makeStages()
    }
}
