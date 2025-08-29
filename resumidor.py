import ollama
import os

MODELO = 'gemma3:4b'
def resumir_tarea(ruta_archivo, nombre_dir:str, imagenes: list[str]|None = None)-> str:
    if not os.path.exists(ruta_archivo):
        return f"Error: El archivo '{ruta_archivo}' no fue encontrado."

    try:
        with open(ruta_archivo, 'r', encoding='utf-8') as archivo:
            contenido_script = archivo.read()

        if not contenido_script.strip():
            return "El archivo está vacío. No hay nada que resumir."

        prompt = f"""
        Dado el siguiente archivo markdown, extrae una lista de tareas o pendientes que se pueden identificar en el contenido. Si no hay tareas, responde vacio.
        si hay tareas, responde con una lista en formato markdown, cada tarea debe empezar con un guión.
        No agregues nada más, solo la lista de tareas.
        No agregues explicaciones ni introducciones, solo la lista de tareas.
        La lista debe ser como la siguiente:
            - [ ] @{{ *fecha de entrega en formato YYYY-MM-DD* }} / *Materia* / *Descripcion*
        Asegúrate de que las fechas de entrega estén en el formato @{{YYYY-MM-DD}} y si no existe una fecha de entrega, asume que la fecha de entrega es el dia siguiente
        El nombre de la materia es {nombre_dir}, si no puedes encontrar una materia, usa "General" como materia.
        Divide la fecha de entrega, la materia y la descripcion con una barra inclinada (/).
        Si no puedes encontrar una materia, usa "General" como materia.


        Fecha actual: {os.popen('date +"%Y-%m-%d"').read().strip()}
        Dia de la semana actual: {os.popen('date +"%A"').read().strip()}
        Nombre del archivo: {os.path.basename(ruta_archivo)}
```markdown
        {contenido_script}
```
        """

        respuesta = ollama.chat(
                model=MODELO,
            messages=[
                {'role': 'system', 'content': 'Eres un asistente útil que dado un archivo de markdown y la fecha actual escribes una lista con las taras o pendientes que se pueden extraer del archivo. La lista debe estar en formato markdown y cada tarea debe empezar con un guión. Si no hay tareas, responde vacio'

                    },
                {
                    'role': 'user',
                    'content': prompt,
                },
            ]
        )

        return respuesta['message']['content']

    except Exception as e:
        return (f"Ha ocurrido un error.\n"
                f"Detalle del error: {e}")

