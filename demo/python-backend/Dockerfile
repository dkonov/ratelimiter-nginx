FROM python:3.13-alpine
WORKDIR /app
COPY clients/python/ratelimiter.py /app/ratelimiter.py
COPY demo/python-backend/app.py /app/app.py
CMD ["python", "/app/app.py"]
