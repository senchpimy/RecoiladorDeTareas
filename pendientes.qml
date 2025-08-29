import Quickshell
import Quickshell.Io
import QtQuick
import Quickshell.Wayland
import QtQuick.Controls
import QtQuick.Layouts
import Qt5Compat.GraphicalEffects

ShellRoot {
    id: main
    property string font: "System Font"


    property string serverUrl: "http://localhost:8080"
    property int pend_n: 0
    property bool themeLoaded: false
    property bool alreadyLoaded: false

    property color panelBackgroundColor: "#FFFFFF"
    property color primaryTextColor: "#000000"
    property color secondaryTextColor: "#666666"
    property color dividerColor: "#CCCCCC"
    property color accentColor: "#FF5722"
    property color uncheckedColor: "#BDBDBD"

    Loader {
        id: themeLoader
        source: "./Colores.qml"
        onLoaded: {
            var colores = themeLoader.item;
            main.themeLoaded = true;
            console.log("Colores loaded:", colores);

            main.primaryTextColor = colores.primaryTextColor;
            main.secondaryTextColor = colores.secondaryTextColor;
            main.dividerColor = colores.dividerColor;
            main.accentColor = colores.accent_color;
            main.uncheckedColor = colores.unchecked_color;

            console.log("Colores asignados correctamente.");
            populateColumnFromServer();
        }
    }

    Timer {
        id: reloadTimer
        interval: main.alreadyLoaded ? 60000*60 : 5000 
        running: true
        repeat: true
        onTriggered: populateColumnFromServer()
    }

    Component {
        id: checkBoxComponent
        Item {
            id: itemRoot
            width: parent.width
            height: content.height + 20

            property alias checked: dynamicCheckBox.checked
            property string dynamicText: "Texto no definido"
            property int dynamicIndex: -1
            property bool dynamicCheck: false

            property var _parsedData: {
                if (typeof dynamicText !== 'string' || !dynamicText) {
                    return { date: "", subject: "", description: "Texto inválido" };
                }
                
                const parts = dynamicText.split(/\s*\/\s*/);
                if (parts.length < 3) {
                    return { date: "", subject: "", description: dynamicText };
                }

                const dateMatch = parts[0].match(/(\d{4}-\d{2}-\d{2})/);
                
                return {
                    date: dateMatch ? dateMatch[1] : "",
                    subject: parts[1],
                    description: parts.slice(2).join(' / ') 
                };
            }

            property string dueDate: _parsedData.date
            property string subject: _parsedData.subject
            property string description: _parsedData.description

            readonly property string daysRemainingText: {
                if (!dueDate) return "";

                const today = new Date();
                today.setHours(0, 0, 0, 0);

                const dateParts = dueDate.split('-');
                const dueDateObj = new Date(dateParts[0], dateParts[1] - 1, dateParts[2]);
                dueDateObj.setHours(0, 0, 0, 0);

                const diffTime = dueDateObj.getTime() - today.getTime();
                const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));

                if (diffDays < 0) {
                    return "Venció hace " + Math.abs(diffDays) + (Math.abs(diffDays) === 1 ? " día" : " días");
                } else if (diffDays === 0) {
                    return "Vence hoy";
                } else if (diffDays === 1) {
                    return "Vence mañana";
                } else {
                    return "Vence en " + diffDays + " días";
                }
            }


            MouseArea {
                anchors.fill: parent
                onClicked: {
                    dynamicCheckBox.checked = !dynamicCheckBox.checked;
                    sendUpdateToServer(dynamicIndex, dynamicCheckBox.checked);
                }
            }

            CheckBox {
                id: dynamicCheckBox
                visible: false
                checked: itemRoot.dynamicCheck
            }

            Rectangle {
                id: indicator
                implicitWidth: 26
                implicitHeight: 26
                x: 15
                y: parent.height / 2 - height / 2
                radius: 13
                border.color: itemRoot.checked ? main.accentColor : main.uncheckedColor
                border.width: 2

                Rectangle {
                    width: 18
                    height: 18
                    anchors.centerIn: parent
                    radius: 9
                    color: main.accentColor
                    visible: itemRoot.checked
                }
            }

            Column {
                id: content
                anchors.left: indicator.right
                anchors.leftMargin: 15
                anchors.right: parent.right
                anchors.rightMargin: 15
                anchors.verticalCenter: parent.verticalCenter
                spacing: 4

                Row {
                    width: parent.width
                    spacing: 8
                    visible: daysRemainingText !== "" || subject !== ""

                    Text {
                        text: daysRemainingText
                        font.family: main.font
                        font.pointSize: 10
                        color: main.secondaryTextColor
                        opacity: itemRoot.checked ? 0.6 : 1.0
                        visible: daysRemainingText !== ""
                    }
                    Text {
                        text: subject
                        font.family: main.font
                        font.pointSize: 10
                        font.bold: true
                        color: main.accentColor
                        opacity: itemRoot.checked ? 0.6 : 1.0
                        visible: subject !== ""
                    }
                }

                Text {
                    text: description
                    color: itemRoot.checked ? main.uncheckedColor : main.primaryTextColor
                    font.family: main.font
                    font.pointSize: 13
                    wrapMode: Text.WordWrap
                    width: parent.width
                    elide: Text.ElideRight
                    font.strikeout: itemRoot.checked
                }
            }
        }
    }

    function populateColumnFromServer() {
        var xhr = new XMLHttpRequest();
        xhr.onreadystatechange = function() {
            if (xhr.readyState === XMLHttpRequest.DONE) {
                if (xhr.status === 200) {
                    var pendientesData;
                    try {
                      pendientesData = JSON.parse(xhr.responseText);
                      main.alreadyLoaded = true;
                    } catch (e) {
                        console.error("Error al parsear JSON:", e, "Respuesta:", xhr.responseText);
                        pendientesData = null;
                    }
                    createCheckboxesFromServerData(pendientesData);
                } else {
                    console.error("Error del servidor:", xhr.status, "Respuesta:", xhr.responseText);
                    createCheckboxesFromServerData(null);
                }
            }
        }
        xhr.open("GET", main.serverUrl + "/pendientes", true);
        xhr.send();
    }

    function sendUpdateToServer(index, isChecked) {
        var xhr = new XMLHttpRequest();
        xhr.open("POST", main.serverUrl + "/update", true);
        xhr.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
        var data = JSON.stringify({ "index": index, "checked": isChecked });
        xhr.send(data);
    }

    function createCheckboxesFromServerData(pendientes) {
        for (var i = column.children.length - 1; i >= 0; i--) {
            if (column.children[i]) {
                column.children[i].destroy();
            }
        }
        main.pend_n = 0;

        if (!pendientes || !Array.isArray(pendientes) || pendientes.length === 0) {
            return;
        }

        pendientes.forEach(function(item, index) {
            if (!item) return;
            if (item.hasOwnProperty('checked') && !item.checked) {
                main.pend_n += 1;
            }
            checkBoxComponent.createObject(column, {
                "dynamicText": item.text || "Texto por defecto",
                "dynamicIndex": index,
                "dynamicCheck": !!item.checked
            });
        });
    }

    PanelWindow {
        id: window
        implicitWidth: 300
        implicitHeight: 450
        color: "transparent"
        anchors.left: true
        WlrLayershell.layer: WlrLayer.Bottom
        
        property var borde: 30
        property var radio: 20

        Rectangle {
            id: mainCard
            width: parent.width - window.borde
            height: parent.height
            x: window.borde
            y: 0
            color: main.panelBackgroundColor
            radius: window.radio

            DropShadow {
                anchors.fill: parent
                radius: 12.0
                samples: 24
                color: "#40000000"
                horizontalOffset: 0
                verticalOffset: 4
            }

            Rectangle {
                anchors.fill: parent
                color: "transparent"
                radius: parent.radius
                clip: true

                ColumnLayout {
                    id: mainLayout
                    width: parent.width
                    height: parent.height
                    spacing: 0

                    Column {
                        id: headerColumn
                        Layout.fillWidth: true
                        Layout.preferredHeight: childrenRect.height
                        z: 1

                        Rectangle { height: 15; width: parent.width; color: "transparent" }

                        RowLayout {
                            width: parent.width

                            Rectangle { height: 10;
                            width: 5
                            color: "transparent"
                            Layout.alignment: Qt.AlignLeft
                          }

                            Text {
                                text: "Pendientes"
                                font.family: main.font
                                font.pointSize: 16
                                font.bold: true
                                color: main.accentColor
                            }

                            Text {
                                text: Number(main.pend_n)
                                font.family: main.font
                                font.pointSize: 16
                                font.bold: true
                                color: main.primaryTextColor 
                                Layout.alignment: Qt.AlignRight
                              }
                            Rectangle { height: 10;
                            width: 5
                            color: "transparent"
                            Layout.alignment: Qt.AlignRight
                          }
                        }

                        Rectangle { height: 10; width: parent.width; color: "transparent" }
                        
                        Rectangle {
                            width: parent.width
                            height: 2
                            color: main.dividerColor
                            anchors.horizontalCenter: parent.horizontalCenter
                        }
                    }

                    Flickable {
                        id: flickable
                        Layout.fillWidth: true
                        Layout.fillHeight: true
                        contentWidth: width
                        contentHeight: column.height
                        clip: true

                        ColumnLayout {
                            id: column
                            width: parent.width
                            spacing: 0
                        }
                    }
                }
            }
        }
    }
}
