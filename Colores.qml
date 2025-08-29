import QtQuick

QtObject {
    property color panelBackgroundColor: "#5cddf1"
    property color primaryTextColor: "#2b3133" // Este no usaba set_lightness en tu ejemplo, se tomaba directo
    property color secondaryTextColor: "#899295"         // Este tampoco usaba set_lightness en tu ejemplo
    property color dividerColor: "#a4abad"
    property color accent_color: "#6ecadf"
    property color unchecked_color: "#969ea1"
}
