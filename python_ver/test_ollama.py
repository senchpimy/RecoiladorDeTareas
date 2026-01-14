import requests
import json
import datetime

url = "http://localhost:11434/api/chat"
model = "gemma3:12b"

system_prompt = """Dado el siguiente archivo markdown, extrae una lista de tareas o pendientes que se pueden identificar en el contenido. Si no hay tareas, responde vacio.
si hay tareas, responde con una lista en formato markdown, cada tarea debe empezar con un guión.
No agregues nada más, solo la lista de tareas.
No agregues explicaciones ni introducciones, solo la lista de tareas.
La lista debe ser como la siguiente:
    - [ ] @{ *fecha de entrega en formato YYYY-MM-DD* } / *Materia* / *Descripcion*
Asegúrate de que las fechas de entrega estén en el formato @{YYYY-MM-DD} y si no existe una fecha de entrega, asume que la fecha de entrega es el dia siguiente
Si no puedes encontrar una materia, usa "General" como materia.
Divide la fecha de entrega, la materia y la descripcion con una barra inclinada (/). 

Si no hay tareas, responde "None"."""

user_content_example = """
# Clase de Redes
2026-01-13

El profesor mencionó que debemos terminar la configuración del router.
- [ ] Configurar VLAN 10
- [ ] Configurar VLAN 20

Tarea para el viernes:
Investigar sobre OSPF y hacer un resumen de 1 cuartilla.
"""

formatted_user_prompt = f"""
        El nombre de la materia es Redes,


        Fecha actual: 2026-01-13
        Dia de la semana actual: Tuesday
        Nombre del archivo: 2026-01-13 Redes.md
```markdown
{user_content_example}
```
"""

payload = {
    "model": model,
    "stream": False,
    "messages": [
        {"role": "system", "content": system_prompt},
        {"role": "user", "content": formatted_user_prompt}
    ]
}

print("Enviando petición a Ollama...")
try:
    response = requests.post(url, json=payload)
    response.raise_for_status()
    result = response.json()
    print("\n--- Respuesta del Modelo ---")
    print(result['message']['content'])
    print("----------------------------")
except Exception as e:
    print(f"Error: {e}")
