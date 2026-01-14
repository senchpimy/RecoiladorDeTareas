import ollama
import os

#MODELO = 'gemma3:4b'
#MODELO = 'gpt-oss'
MODELO = 'gemma3:12b'
def generar_ejemplo_tarea()-> str:
    prompt = """
    El nombre de la materia es, Internet of Things,


    Fecha actual: 2023-08-28
    Dia de la semana actual: Jueves
    Nombre del archivo: "Sistemas de Informacion 2023-08-27.md"
```markdown
Construir una cerradura combinacional
Poner 8 entradas y hacer valer 5 digitos y que cuando se presione enter verificar que la contrasena sea correcta

Para el prox miercoles

Dado que esta es una practica formal se tiene que entregar una documentacion en pdf y en el classroom


El pdf va a llevar el nombre en la parte derecha superior y el numero de practica y circuito, no lleva portada.

Una introduccion donde se especifique el circuito que se hizo, un diagrama de bloques, un diagrama electrico, el codigo fuente y el circuito funcionando
```
    """
    return prompt


def generar_ejemplo_tareas_vacias()-> str:
    prompt = """
    El nombre de la materia es Sistemas de Informacion,


    Fecha actual: 2023-10-10
    Dia de la semana actual: Lunes
    Nombre del archivo: "Sistemas de Informacion 2023-10-09.md"
```markdown
El scrum team son los que van a leer el backlog y van a actuar de forma acorde

### Historia de un usuario 

Es una representacion de un requisito escrito en una o dos frases utilizando el lenguaje comun del usuario

Son utilizadas para la especificacion de requisitos agiles (acompanadas de las discusiones con los usuarios y las pruebas de validacion)

Cada historia de usuario debe ser limitada, etsa deberia poderse escribir sobre una nora adhesiva pequena

- Centran la atencion en el usuario
- Permiten la colaboracion

Deben de ser escritas en una tarjeta
deben de ser conversadas y debn de ser confirmadas
```
    """
    return prompt


def resumir_tarea(ruta_archivo, nombre_dir:str, imagenes: list[str]|None = None)-> str:
    if not os.path.exists(ruta_archivo):
        return f"Error: El archivo '{ruta_archivo}' no fue encontrado."

    try:
        with open(ruta_archivo, 'r', encoding='utf-8') as archivo:
            contenido_script = archivo.read()

        if not contenido_script.strip():
            return "None"

        prompt = f"""
        El nombre de la materia es {nombre_dir},


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
                {'role': 'system', 'content': """Dado el siguiente archivo markdown, extrae una lista de tareas o pendientes que se pueden identificar en el contenido. Si no hay tareas, responde vacio.
si hay tareas, responde con una lista en formato markdown, cada tarea debe empezar con un guión.
No agregues nada más, solo la lista de tareas.
No agregues explicaciones ni introducciones, solo la lista de tareas.
La lista debe ser como la siguiente:
    - [ ] @{{ *fecha de entrega en formato YYYY-MM-DD* }} / *Materia* / *Descripcion*
Asegúrate de que las fechas de entrega estén en el formato @{{YYYY-MM-DD}} y si no existe una fecha de entrega, asume que la fecha de entrega es el dia siguiente
Si no puedes encontrar una materia, usa "General" como materia.
Divide la fecha de entrega, la materia y la descripcion con una barra inclinada (/).

Si no hay tareas, responde "None"."""
                    },
                #Tarea vacia
                {'role': 'user',
                 'content': generar_ejemplo_tareas_vacias()
                    },
                {
                    'role': "assistant"
                    , 'content': "None"
                    },
                #Tarea con contenido
                {'role': 'user',
                 'content': generar_ejemplo_tareas_vacias()
                    },
                {
                    'role': "assistant"
                    , 'content': "- [ ] @{2025-08-31} / Internet of Things / Construir una cerradura combinacional con 8 entradas y 5 digitos, verificar la contraseña al presionar enter, preparar documentación en PDF (incluyendo circuito, diagrama de bloques, diagrama eléctrico, código fuente y circuito funcionando)"
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

