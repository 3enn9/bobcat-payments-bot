import io
import os
import ssl
import time

import pdfplumber
from io import BytesIO
import re
import fitz
from telegram import Bot
from dotenv import load_dotenv

import certifi
from imapclient import IMAPClient
import pyzmail
from telegram.error import RetryAfter

HOST = 'imap.yandex.ru'

ssl_context = ssl.create_default_context(cafile=certifi.where())

load_dotenv()

BOT_TOKEN = os.getenv("BOT_TOKEN")
CHAT_ID = os.getenv("CHAT_ID")
USERNAME = os.getenv("USERNAME")
PASSWORD = os.getenv("PASSWORD")
bot = Bot(token=BOT_TOKEN)

def process_message(server, uid):
    raw_message = server.fetch([uid], ['BODY.PEEK[]', 'FLAGS'])
    message = pyzmail.PyzMessage.factory(raw_message[uid][b'BODY[]'])

    from_ = message.get_addresses('from')

    if from_[0][1] != "ratavina@mail.ru":
        return

    for part in message.mailparts:
        if part.filename and part.filename.endswith(".pdf") and ("сч " in part.filename or "Сч " in part.filename or "Акт сверки" in part.filename):

            pdf_bytes = part.get_payload()

            pdf_file = BytesIO(pdf_bytes)

            with pdfplumber.open(pdf_file) as pdf:
                text = ""
                for page in pdf.pages:
                    text += page.extract_text() or ""

            check = parse_invoice(text)
            result = os.path.splitext(part.filename)[0].strip('"') + "\n" + check
            if "Акт сверки" in part.filename:
                send_pdf_as_images(pdf_bytes, part.filename, "акт")
            elif "сст" in part.filename:
                send_pdf_as_images(pdf_bytes, result, "сст")
            elif "анн" in part.filename:
                send_pdf_as_images(pdf_bytes, result, "анн")
            elif "анв" in part.filename:
                send_pdf_as_images(pdf_bytes, result, "анв")
            elif "адн" in part.filename:
                send_pdf_as_images(pdf_bytes, result, "адн")
            elif "сан" in part.filename:
                send_pdf_as_images(pdf_bytes, result, "сан")

            print(uid, result)

def idle_loop():
    last_uid = 58518
    processed = set()

    while True:
        try:
            with IMAPClient(HOST, ssl=True) as server:
                server.login(USERNAME, PASSWORD)
                server.select_folder('INBOX')

                # print("✅ подключено")

                while True:
                    server.idle()

                    server.idle_check(timeout=60)
                    server.idle_done()

                    messages = server.search(['UID', f'{last_uid + 1}:*'])

                    if messages:
                        for uid in messages:
                            if uid in processed:
                                continue

                            processed.add(uid)

                            if uid > last_uid:
                                process_message(server, uid)

                        last_uid = max(messages)

                    time.sleep(1)

        except Exception as e:
            print(f"⚠️ ОШИБКА: {type(e).__name__}: {e}", flush=True)
            import traceback
            traceback.print_exc()
            print("🔄 Переподключение через 5 секунд...", flush=True)
            time.sleep(5)

def parse_invoice(text: str):
    buyer = re.search(
        r"Покупатель\s+(.*?)(?=\s*,?\s*ИНН)",
        text,
        re.S
    )

    if buyer:
        buyer = re.sub(r'\s+', ' ', buyer.group(1)).strip()

    if buyer is None:
        return ""
    return buyer


def send_pdf_as_images(pdf_bytes, text, org):
    orgMap = {"сст": 1221, "анн": 1223, "анв": 1227, "адн": 1233, "сан": 1237, "акт": 1657}

    doc = fitz.open(stream=pdf_bytes, filetype="pdf")

    for i, page in enumerate(doc):
        pix = page.get_pixmap()
        img_bytes = pix.tobytes("png")

        bio = io.BytesIO(img_bytes)
        bio.name = f"page_{i}.png"
        bio.seek(0)

        while True:
            try:
                bot.send_photo(
                    chat_id=CHAT_ID,
                    photo=bio,
                    caption=text if i == 0 else None,
                    message_thread_id=orgMap[org]
                )

                time.sleep(1)
                break

            except RetryAfter as e:
                print(f"FloodWait {e.retry_after}")
                time.sleep(e.retry_after + 1)

if __name__ == "__main__":
    idle_loop()