from fastapi import FastAPI
from pydantic import BaseModel
from sentence_transformers import SentenceTransformer
from transformers import pipeline

app = FastAPI()
embedder = SentenceTransformer("all-MiniLM-L6-v2")
zero_shot = pipeline("zero-shot-classification", model="facebook/bart-large-mnli")

class EmbedRequest(BaseModel):
    text: str

class EmbedResponse(BaseModel):
    embedding: list[float]


class ClassifyRequest(BaseModel):
    text: str


class ClassifyResponse(BaseModel):
    rating: str

@app.post("/embed", response_model=EmbedResponse)
def embed(req: EmbedRequest):
    vec = embedder.encode(req.text, normalize_embeddings=True)
    return {"embedding": vec.tolist()}


@app.post("/classify", response_model=ClassifyResponse)
def classify(req: ClassifyRequest):
    labels = ["left", "center", "right"]
    result = zero_shot(req.text, labels)
    return {"rating": result["labels"][0]}
