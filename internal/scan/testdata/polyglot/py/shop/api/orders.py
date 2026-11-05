from fastapi import APIRouter
from . import repo
router = APIRouter(prefix="/orders")

@router.get("/{order_id}")
def get(order_id: int):
    return repo.load(order_id)
