import os
import re
from datetime import datetime, timedelta
import resumidor

DIRECTORIO_POR_DEFECTO = "/home/plof/Documents/Actual/actual"
DIAS_A_REVISAR = 7
METADATA = "---\nprocesado_por_ia: true\n---\n\n"
TESTING = True


def es_apunte_reciente_y_sin_procesar(ruta_archivo: str) -> bool:
    """
    Verifica si un archivo es un apunte de Markdown, tiene una fecha reciente
    en su nombre y no ha sido procesado previamente por la IA.
    """
    dias_max = DIAS_A_REVISAR
    nombre_archivo = os.path.basename(ruta_archivo)

    if not nombre_archivo.endswith(".md"):
        return False

    coincidencia = re.search(r"(\d{4}-\d{2}-\d{2})", nombre_archivo)
    if not coincidencia:
        return False

    fecha_archivo_str = coincidencia.group(1)
    try:
        fecha_archivo = datetime.strptime(fecha_archivo_str, "%Y-%m-%d")
        if datetime.now() - fecha_archivo > timedelta(days=dias_max):
            return False  # El archivo es demasiado antiguo.
    except ValueError:
        return False  # La fecha en el nombre no es válida.

    try:
        with open(ruta_archivo, "r", encoding="utf-8") as f:
            contenido = f.read()
            if "procesado_por_ia: true" in contenido:  # Mejorar
                return False
    except IOError:
        return False

    return True


def obtener_archivos_a_procesar(directorio: str) -> list[str]:
    """
    Escanea un directorio y devuelve una lista de rutas de archivo que son
    apuntes recientes y que aún no han sido procesados.
    """
    archivos_pendientes = []
    for nombre_archivo in sorted(os.listdir(directorio)):
        ruta_completa = os.path.join(directorio, nombre_archivo)
        if os.path.isfile(ruta_completa):
            if es_apunte_reciente_y_sin_procesar(ruta_completa):
                archivos_pendientes.append(ruta_completa)
    return archivos_pendientes


def procesar_y_sellar_archivo(ruta_archivo: str, nombre_dir: str) -> str:
    print(f"Procesando: {os.path.basename(ruta_archivo)}...")

    resumen = resumidor.resumir_tarea(ruta_archivo, nombre_dir)

    try:
        with open(ruta_archivo, "r+", encoding="utf-8") as f:
            contenido_original = f.read()
            f.seek(0, 0)
            if not TESTING:
                f.write(METADATA + contenido_original)
    except IOError as e:
        print(e)

    return resumen


def generar_informe(informes_procesados: list[str], directorio: str):
    """
    Imprime en la consola el informe final consolidado a partir de los resúmenes.
    """
    print("  INFORME DE NUEVAS TAREAS Y DATOS\n")

    if informes_procesados:
        print("\n\n".join(informes_procesados))
    else:
        print(f"No se encontraron apuntes nuevos para procesar en '{directorio}'.")


def main():
    directorio_base = DIRECTORIO_POR_DEFECTO

    if not os.path.isdir(directorio_base):
        print(f"Error: El directorio base '{directorio_base}' no fue encontrado.")
        exit(1)

    informe_final_total = []

    for nombre_materia in sorted(os.listdir(directorio_base)):
        ruta_materia = os.path.join(directorio_base, nombre_materia)

        if os.path.isdir(ruta_materia):
            print(f"Escaneando materia: {nombre_materia}")

            archivos_a_procesar = obtener_archivos_a_procesar(ruta_materia)

            if not archivos_a_procesar:
                print("No hay apuntes nuevos para procesar.\n")
                continue

            for ruta_archivo in archivos_a_procesar:
                resumen = procesar_y_sellar_archivo(ruta_archivo, nombre_materia)

                nombre_base = os.path.basename(ruta_archivo)
                informe_final_total.append(
                    f"Resumen [{nombre_materia}]: {nombre_base}\n{resumen}"
                )

    generar_informe(informe_final_total, directorio_base)


if __name__ == "__main__":
    main()
