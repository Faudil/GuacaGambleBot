from src.items.Item import Item, ItemRarity, ItemType


class ResourceItem(Item):
    def __init__(self, name, price, desc, rarity=ItemRarity.common, pet_effet: dict =None):
        super().__init__(name, price, desc, rarity=rarity, item_type=ItemType.resource, pet_effet=pet_effet)

    async def use(self, ctx, **kwargs):
        await ctx.send(f"🤔 Tu regardes **{self.name}**... C'est joli, mais ça ne fait rien.")
        return False
